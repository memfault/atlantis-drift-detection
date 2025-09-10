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

// FormatPlanDriftMessageWithDetails formats a plan drift message for Slack with Memfault-specific command and drift details
// This method extracts the drift details from the Terraform output and includes them in the notification
func (m *MemfaultSlackFormatter) FormatPlanDriftMessageWithDetails(dir string, terraformOutput string) (string, error) {
	// Extract the environment from the path (e.g., "eu" -> "eu", "staging" -> "staging", "production" -> "prod")
	environment, err := m.extractEnvironment(dir)
	if err != nil {
		return "", fmt.Errorf("failed to format plan drift message: %w", err)
	}

	// Build the profile name from the environment
	profile := m.buildProfile(environment)

	// Extract the project name from the last part of the directory
	project := m.extractProject(dir)

	// Extract the drift details from the Terraform output
	driftDetails := m.extractDriftDetails(terraformOutput)

	message := fmt.Sprintf("Terraform drift detected in `%s`.\n", dir)
	message += "Fix locally with this command:\n\n"
	message += fmt.Sprintf("```\naws-vault exec %s -- inv terraform.apply -p %s\n```\n\n", profile, project)

	if driftDetails != "" {
		message += "Drift Details:\n"
		message += fmt.Sprintf("```\n%s\n```", driftDetails)
	}

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

// extractDriftDetails extracts the drift details from Terraform output
// Slack limits messages to 4k. Therefore, returns the first 1000 characters,
// then "...", then the Plan section.
// If less than 1000 characters total, includes all of it.
func (m *MemfaultSlackFormatter) extractDriftDetails(terraformOutput string) string {
	startMarker := "Terraform will perform the following actions:"
	endMarker := "Plan:"

	startIndex := strings.Index(terraformOutput, startMarker)
	if startIndex == -1 {
		return ""
	}

	// Move to the end of the start marker
	startIndex += len(startMarker)

	// Find the Plan section (including the newline after "Plan:")
	planIndex := strings.Index(terraformOutput[startIndex:], endMarker)
	if planIndex == -1 {
		// If no Plan section found, return everything after start marker
		extracted := strings.TrimSpace(terraformOutput[startIndex:])
		if len(extracted) > 1000 {
			return m.truncateAtNewline(extracted, 1000)
		}
		return extracted
	}

	// Find the end of the Plan line (including the newline)
	planStart := startIndex + planIndex
	planLineEnd := strings.Index(terraformOutput[planStart:], "\n")
	if planLineEnd == -1 {
		// If no newline after Plan, include everything to the end
		planSection := terraformOutput[planStart:]
		driftContent := strings.TrimSpace(terraformOutput[startIndex:planStart])

		if len(driftContent) <= 1000 {
			return driftContent + "\n" + planSection
		}

		return driftContent[:1000] + "\n...\n" + planSection
	}

	planSection := terraformOutput[planStart : planStart+planLineEnd]
	driftContent := strings.TrimSpace(terraformOutput[startIndex:planStart])

	// If drift content is less than or equal to 1000 characters, include all of it
	if len(driftContent) <= 1000 {
		return driftContent + "\n" + planSection
	}

	// Otherwise, truncate to first 1000 characters and add "..."
	return driftContent[:1000] + "\n...\n" + planSection
}

// truncateAtNewline truncates text at the first newline after the specified length
func (m *MemfaultSlackFormatter) truncateAtNewline(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Find the first newline after maxLength
	searchStart := maxLength
	newlineIndex := strings.Index(text[searchStart:], "\n")

	if newlineIndex == -1 {
		// No newline found after maxLength, truncate at maxLength
		return text[:maxLength]
	}

	// Return text up to and including the newline
	return text[:searchStart+newlineIndex+1]
}
