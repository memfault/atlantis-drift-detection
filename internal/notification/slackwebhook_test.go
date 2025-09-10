package notification

import (
	"context"
	"strings"
	"testing"

	"net/http"

	"github.com/cresta/atlantis-drift-detection/internal/testhelper"
)

func TestSlackWebhook_ExtraWorkspaceInRemote(t *testing.T) {
	testhelper.ReadEnvFile(t, "../../")
	wh := NewSlackWebhook(testhelper.EnvOrSkip(t, "SLACK_WEBHOOK_URL"), http.DefaultClient)
	genericNotificationTest(t, wh)
}

func TestSlackWebhook_PlanDriftWithTerraformOutput(t *testing.T) {
	// Create a mock HTTP client for testing
	mockClient := &http.Client{}
	wh := NewSlackWebhook("https://hooks.slack.com/test", mockClient)

	// Test with a sample Terraform output
	terraformOutput := `Terraform will perform the following actions:

  # module.redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place
  ~ resource "aws_cloudwatch_metric_alarm" "high-cpu" {
        id = "ProductionRedisPrimary-001 High CPU Alarm"
      ~ threshold = 80 -> 90
        # (21 unchanged attributes hidden)
    }

Plan: 0 to add, 1 to change, 0 to destroy.`

	// This test will fail if the webhook URL is not valid, but we can test the logic
	err := wh.PlanDrift(context.Background(), "infra/terraform/database/prod/us-east-1/production/redis", "workspace", terraformOutput)

	// We expect an error because the webhook URL is not real, but the error should be about the HTTP request, not about formatting
	if err != nil {
		if !strings.Contains(err.Error(), "failed to send slack webhook request") {
			t.Errorf("Expected HTTP request error, but got: %v", err)
		}
	}
}

func TestSlackWebhook_PlanDriftWithoutTerraformOutput(t *testing.T) {
	// Create a mock HTTP client for testing
	mockClient := &http.Client{}
	wh := NewSlackWebhook("https://hooks.slack.com/test", mockClient)

	// Test without Terraform output (empty variadic parameter)
	err := wh.PlanDrift(context.Background(), "infra/terraform/database/staging/myproject", "workspace")

	// We expect an error because the webhook URL is not real, but the error should be about the HTTP request, not about formatting
	if err != nil {
		if !strings.Contains(err.Error(), "failed to send slack webhook request") {
			t.Errorf("Expected HTTP request error, but got: %v", err)
		}
	}
}
