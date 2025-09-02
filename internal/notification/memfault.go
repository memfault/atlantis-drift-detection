package notification

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MemfaultSlackFormatter formats Slack messages specifically for Memfault's workflow
type MemfaultSlackFormatter struct{}

// NewMemfaultSlackFormatter creates a new instance of MemfaultSlackFormatter
func NewMemfaultSlackFormatter() *MemfaultSlackFormatter {
	return &MemfaultSlackFormatter{}
}

// FormatPlanDriftMessage formats a plan drift message for Slack with Memfault-specific command
func (m *MemfaultSlackFormatter) FormatPlanDriftMessage(dir string) (string, error) {
	// Extract the environment from the path (e.g., "eu" -> "eu", "staging" -> "staging", "production" -> "prod")
	environment, err := m.extractEnvironment(dir)
	if err != nil {
		return "", fmt.Errorf("failed to format plan drift message: %w", err)
	}

	// Build the profile name from the environment
	profile := m.buildProfile(environment)

	// Extract the project name from the last part of the directory
	project := m.extractProject(dir)

	message := fmt.Sprintf("Terraform drift detected in `%s`.\n", dir)
	message += "Fix locally with this command:\n\n"
	message += fmt.Sprintf("```\naws-vault exec %s -- inv terraform.apply -p %s\n```", profile, project)

	return message, nil
}

// extractEnvironment extracts the environment from the directory path
// Example: "infra/terraform/database/prod/eu-central-1/eu/lavinmq" -> "eu"
// Example: "infra/terraform/database/staging/lavinmq" -> "staging"
// Example: "infra/terraform/database/production/lavinmq" -> "production"
func (m *MemfaultSlackFormatter) extractEnvironment(dir string) (string, error) {
	parts := strings.Split(dir, "/")

	// Look for the environment part (second to last part)
	if len(parts) >= 2 {
		environment := parts[len(parts)-2]
		return environment, nil
	}

	// Error if we can't extract the environment
	return "", fmt.Errorf("cannot extract environment from directory path: %s (need at least 2 path components)", dir)
}

// buildProfile builds the profile name from the environment
// Example: "eu" -> "memfault-eu", "staging" -> "memfault-staging", "production" -> "memfault-prod"
func (m *MemfaultSlackFormatter) buildProfile(environment string) string {
	// Special case for production
	if environment == "production" {
		return "memfault-prod"
	}

	return "memfault-" + environment
}

// extractProject extracts the project name from the last part of the directory
// Example: "infra/terraform/database/prod/eu-central-1/eu/lavinmq" -> "lavinmq"
func (m *MemfaultSlackFormatter) extractProject(dir string) string {
	return filepath.Base(dir)
}
