package notification

import (
	"strings"
	"testing"
)

func TestMemfaultSlackFormatter_FormatPlanDriftMessage(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		expected    string
		expectError bool
	}{
		{
			name:        "eu environment with lavinmq project",
			dir:         "infra/terraform/database/prod/eu-central-1/eu/lavinmq",
			expected:    "Terraform drift detected in `infra/terraform/database/prod/eu-central-1/eu/lavinmq`.\nFix locally with this command:\n\n```\naws-vault exec memfault-eu -- inv terraform.apply -p lavinmq\n```",
			expectError: false,
		},
		{
			name:        "staging environment with different project",
			dir:         "infra/terraform/database/staging/myproject",
			expected:    "Terraform drift detected in `infra/terraform/database/staging/myproject`.\nFix locally with this command:\n\n```\naws-vault exec memfault-staging -- inv terraform.apply -p myproject\n```",
			expectError: false,
		},
		{
			name:        "production environment with testapp project",
			dir:         "terraform/infra/production/testapp",
			expected:    "Terraform drift detected in `terraform/infra/production/testapp`.\nFix locally with this command:\n\n```\naws-vault exec memfault-prod -- inv terraform.apply -p testapp\n```",
			expectError: false,
		},
		{
			name:        "simple path structure",
			dir:         "infra/terraform/simple/project",
			expected:    "Terraform drift detected in `infra/terraform/simple/project`.\nFix locally with this command:\n\n```\naws-vault exec memfault-simple -- inv terraform.apply -p project\n```",
			expectError: false,
		},
		{
			name:        "invalid path - only one component",
			dir:         "lavinmq",
			expected:    "",
			expectError: true,
		},
	}

	formatter := NewMemfaultSlackFormatter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.FormatPlanDriftMessage(tt.dir)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("FormatPlanDriftMessage() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestMemfaultSlackFormatter_ExtractEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		expected    string
		expectError bool
	}{
		{"eu environment", "infra/terraform/database/prod/eu-central-1/eu/lavinmq", "eu", false},
		{"staging environment", "infra/terraform/database/staging/myproject", "staging", false},
		{"production environment", "terraform/infra/production/testapp", "production", false},
		{"simple path", "infra/terraform/simple/project", "simple", false}, // second to last part
		{"invalid path - only one component", "lavinmq", "", true},
	}

	formatter := NewMemfaultSlackFormatter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.extractEnvironment(tt.dir)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("extractEnvironment() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestMemfaultSlackFormatter_BuildProfile(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    string
	}{
		{"eu environment", "eu", "memfault-eu"},
		{"staging environment", "staging", "memfault-staging"},
		{"production environment", "production", "memfault-prod"},
	}

	formatter := NewMemfaultSlackFormatter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.buildProfile(tt.environment)
			if result != tt.expected {
				t.Errorf("buildProfile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMemfaultSlackFormatter_ExtractProject(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{"lavinmq", "infra/terraform/database/prod/eu-central-1/eu/lavinmq", "lavinmq"},
		{"myproject", "infra/terraform/database/staging/myproject", "myproject"},
		{"testapp", "terraform/infra/production/testapp", "testapp"},
		{"simple", "infra/terraform/simple", "simple"},
	}

	formatter := NewMemfaultSlackFormatter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.extractProject(tt.dir)
			if result != tt.expected {
				t.Errorf("extractProject() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMemfaultSlackFormatter_MessageStructure(t *testing.T) {
	formatter := NewMemfaultSlackFormatter()
	dir := "infra/terraform/database/prod/eu-central-1/eu/lavinmq"

	result, err := formatter.FormatPlanDriftMessage(dir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that the message contains the expected components
	if !strings.Contains(result, "Terraform drift detected in") {
		t.Error("Message should contain 'Terraform drift detected in'")
	}

	if !strings.Contains(result, "Fix locally with this command:") {
		t.Error("Message should contain 'Fix locally with this command:'")
	}

	if !strings.Contains(result, "aws-vault exec memfault-eu -- inv terraform.apply -p lavinmq") {
		t.Error("Message should contain the correct aws-vault command")
	}

	if !strings.Contains(result, "`infra/terraform/database/prod/eu-central-1/eu/lavinmq`") {
		t.Error("Message should contain the directory in backticks")
	}
}
