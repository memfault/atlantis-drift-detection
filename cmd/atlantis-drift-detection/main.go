package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cresta/atlantis-drift-detection/internal/atlantis"
	"github.com/cresta/atlantis-drift-detection/internal/drifter"
	"github.com/cresta/atlantis-drift-detection/internal/notification"
	"github.com/cresta/atlantis-drift-detection/internal/processedcache"
	"github.com/cresta/atlantis-drift-detection/internal/terraform"
	"github.com/cresta/gogit"
	"github.com/cresta/gogithub"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	// Empty import allows pinning to version atlantis uses
	_ "time/tzdata" // Embed timezone data for containerized environments

	"github.com/joeshaw/envdecode"
	_ "github.com/nlopes/slack"
	"go.uber.org/zap"
)

type config struct {
	Repo               string        `env:"REPO,required"`
	AtlantisHostname   string        `env:"ATLANTIS_HOST,required"`
	AtlantisToken      string        `env:"ATLANTIS_TOKEN,required"`
	AtlantisConfigPath string        `env:"ATLANTIS_CONFIG_PATH,default=atlantis.yaml"`
	DirectoryWhitelist []string      `env:"DIRECTORY_WHITELIST"`
	SlackWebhookURL    string        `env:"SLACK_WEBHOOK_URL"`
	SkipWorkspaceCheck bool          `env:"SKIP_WORKSPACE_CHECK"`
	ParallelRuns       int           `env:"PARALLEL_RUNS"`
	DynamodbTable      string        `env:"DYNAMODB_TABLE"`
	CacheValidDuration time.Duration `env:"CACHE_VALID_DURATION,default=24h"`
	WorkflowOwner      string        `env:"WORKFLOW_OWNER"`
	WorkflowRepo       string        `env:"WORKFLOW_REPO"`
	WorkflowId         string        `env:"WORKFLOW_ID"`
	WorkflowRef        string        `env:"WORKFLOW_REF"`
}

func loadEnvIfExists() error {
	_, err := os.Stat(".env")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error checking for .env file: %v", err)
	}
	return godotenv.Load()
}

type zapGogitLogger struct {
	logger *zap.Logger
}

func (z *zapGogitLogger) build(strings map[string]string, ints map[string]int64) *zap.Logger {
	l := z.logger
	for k, v := range strings {
		l = l.With(zap.String(k, v))
	}
	for k, v := range ints {
		l = l.With(zap.Int64(k, v))
	}
	return l
}

func (z *zapGogitLogger) Debug(_ context.Context, msg string, strings map[string]string, ints map[string]int64) {
	z.build(strings, ints).Debug(msg)
}

func (z *zapGogitLogger) Info(_ context.Context, msg string, strings map[string]string, ints map[string]int64) {
	z.build(strings, ints).Info(msg)
}

var _ gogit.Logger = (*zapGogitLogger)(nil)

func main() {
	ctx := context.Background()
	zapCfg := zap.NewProductionConfig()

	// Respect LOG_LEVEL environment variable, default to info level
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		var level zap.AtomicLevel
		if err := level.UnmarshalText([]byte(logLevel)); err != nil {
			panic(fmt.Errorf("invalid LOG_LEVEL: %s", logLevel))
		}
		zapCfg.Level = level
	} else {
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := zapCfg.Build(zap.AddCaller())
	if err != nil {
		panic(err)
	}
	if err := loadEnvIfExists(); err != nil {
		logger.Panic("Failed to load .env", zap.Error(err))
	}
	var cfg config
	if err := envdecode.Decode(&cfg); err != nil {
		logger.Panic("failed to decode config", zap.Error(err))
	}
	cloner := &gogit.Cloner{
		Logger: &zapGogitLogger{logger},
	}
	notif := &notification.Multi{
		Notifications: []notification.Notification{
			&notification.Zap{Logger: logger.With(zap.String("notification", "true"))},
		},
	}
	if slackClient := notification.NewSlackWebhook(cfg.SlackWebhookURL, http.DefaultClient); slackClient != nil {
		logger.Info("setting up slack webhook notification")
		notif.Notifications = append(notif.Notifications, slackClient)
	}
	var existingConfig *gogithub.NewGQLClientConfig
	if os.Getenv("GITHUB_TOKEN") != "" {
		existingConfig = &gogithub.NewGQLClientConfig{Token: os.Getenv("GITHUB_TOKEN")}
	}
	ghClient, err := gogithub.NewGQLClient(ctx, logger, existingConfig)
	if err != nil {
		logger.Panic("failed to create github client", zap.Error(err))
	}
	if workflowClient := notification.NewWorkflow(ghClient, cfg.WorkflowOwner, cfg.WorkflowRepo, cfg.WorkflowId, cfg.WorkflowRef); workflowClient != nil {
		logger.Info("setting up workflow notification")
		notif.Notifications = append(notif.Notifications, workflowClient)
	}
	tf := terraform.Client{
		Logger: logger.With(zap.String("terraform", "true")),
	}

	var cache processedcache.ProcessedCache = processedcache.Noop{}
	if cfg.DynamodbTable != "" {
		logger.Info("setting up dynamodb result cache")
		cache, err = processedcache.NewDynamoDB(ctx, cfg.DynamodbTable)
		if err != nil {
			logger.Panic("failed to create dynamodb result cache", zap.Error(err))
		}
	}

	d := drifter.Drifter{
		DirectoryWhitelist: cfg.DirectoryWhitelist,
		Logger:             logger.With(zap.String("drifter", "true")),
		Repo:               cfg.Repo,
		AtlantisConfigPath: cfg.AtlantisConfigPath,
		AtlantisClient: &atlantis.Client{
			AtlantisHostname: cfg.AtlantisHostname,
			Token:            cfg.AtlantisToken,
			HTTPClient:       http.DefaultClient,
			Logger:           logger.With(zap.String("atlantis", "true")),
		},
		ParallelRuns:       cfg.ParallelRuns,
		ResultCache:        cache,
		Cloner:             cloner,
		GithubClient:       ghClient,
		CacheValidDuration: cfg.CacheValidDuration,
		Terraform:          &tf,
		Notification:       notif,
		SkipWorkspaceCheck: cfg.SkipWorkspaceCheck,
	}

	logger.Info("Starting drift detection scheduler - will run at 9am ET daily")

	et, err := time.LoadLocation("America/New_York")
	if err != nil {
		logger.Error("Failed to load Eastern Time zone, using local time", zap.Error(err))
		et = time.Local
	}

	c := cron.New(cron.WithLocation(et))

	_, err = c.AddFunc("0 9 * * *", func() {
		logger.Info("Running scheduled drift detection")
		if err := d.Drift(ctx); err != nil {
			logger.Error("Drift detection failed", zap.Error(err))
		} else {
			logger.Info("Drift detection completed successfully")
		}
	})
	if err != nil {
		logger.Panic("Failed to schedule drift detection", zap.Error(err))
	}

	c.Start()
	logger.Info("Cron scheduler started")

	select {}
}
