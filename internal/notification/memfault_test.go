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

func TestMemfaultSlackFormatter_FormatPlanDriftMessageWithDetails(t *testing.T) {
	tests := []struct {
		name             string
		dir              string
		terraformOutput  string
		expectedContains []string
		expectError      bool
	}{
		{
			name:            "eu environment with drift details",
			dir:             "infra/terraform/database/prod/eu-central-1/eu/lavinmq",
			terraformOutput: "Terraform will perform the following actions:\n\n  # aws_instance.example will be updated in-place\n  ~ resource \"aws_instance\" \"example\" {\n        id = \"i-1234567890abcdef0\"\n      ~ instance_type = \"t2.micro\" -> \"t2.small\"\n        # (10 unchanged attributes hidden)\n    }\n\nPlan: 0 to add, 1 to change, 0 to destroy.",
			expectedContains: []string{
				"Terraform drift detected in `infra/terraform/database/prod/eu-central-1/eu/lavinmq`",
				"aws-vault exec memfault-eu -- inv terraform.apply -p lavinmq",
				"Drift Details:",
				"aws_instance.example will be updated in-place",
				"instance_type = \"t2.micro\" -> \"t2.small\"",
			},
			expectError: false,
		},
		{
			name:            "staging environment with no drift details",
			dir:             "infra/terraform/database/staging/myproject",
			terraformOutput: "No changes. Your infrastructure matches the configuration.",
			expectedContains: []string{
				"Terraform drift detected in `infra/terraform/database/staging/myproject`",
				"aws-vault exec memfault-staging -- inv terraform.apply -p myproject",
			},
			expectError: false,
		},
		{
			name:            "production environment with complex drift",
			dir:             "terraform/infra/production/testapp",
			terraformOutput: "Terraform will perform the following actions:\n\n  # module.redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id = \"ProductionRedisPrimary-001 High CPU Alarm\"\n      ~ threshold = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.redis.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id = \"ProductionRedisPrimary-002 High CPU Alarm\"\n      ~ threshold = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\nPlan: 0 to add, 2 to change, 0 to destroy.",
			expectedContains: []string{
				"Terraform drift detected in `terraform/infra/production/testapp`",
				"aws-vault exec memfault-prod -- inv terraform.apply -p testapp",
				"Drift Details:",
				"module.redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place",
				"threshold = 80 -> 90",
			},
			expectError: false,
		},
		{
			name:             "invalid path - only one component",
			dir:              "lavinmq",
			terraformOutput:  "Some output",
			expectedContains: []string{},
			expectError:      true,
		},
	}

	formatter := NewMemfaultSlackFormatter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.FormatPlanDriftMessageWithDetails(tt.dir, tt.terraformOutput)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				for _, expected := range tt.expectedContains {
					if !strings.Contains(result, expected) {
						t.Errorf("FormatPlanDriftMessageWithDetails() should contain '%s', but got: %s", expected, result)
					}
				}
			}
		})
	}
}

func TestMemfaultSlackFormatter_ExtractDriftDetails(t *testing.T) {
	tests := []struct {
		name                string
		terraformOutput     string
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:            "standard terraform plan output",
			terraformOutput: "Terraform will perform the following actions:\n\n  # aws_instance.example will be updated in-place\n  ~ resource \"aws_instance\" \"example\" {\n        id = \"i-1234567890abcdef0\"\n      ~ instance_type = \"t2.micro\" -> \"t2.small\"\n        # (10 unchanged attributes hidden)\n    }\n\nPlan: 0 to add, 1 to change, 0 to destroy.",
			expectedContains: []string{
				"aws_instance.example will be updated in-place",
				"instance_type = \"t2.micro\" -> \"t2.small\"",
			},
			expectedNotContains: []string{
				"Terraform will perform the following actions:",
			},
		},
		{
			name:                "no changes output",
			terraformOutput:     "No changes. Your infrastructure matches the configuration.",
			expectedContains:    []string{},
			expectedNotContains: []string{},
		},
		{
			name:            "complex multi-resource changes",
			terraformOutput: "Terraform will perform the following actions:\n\n  # module.redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id = \"ProductionRedisPrimary-001 High CPU Alarm\"\n      ~ threshold = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.redis.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id = \"ProductionRedisPrimary-002 High CPU Alarm\"\n      ~ threshold = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\nPlan: 0 to add, 2 to change, 0 to destroy.",
			expectedContains: []string{
				"module.redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place",
				"module.redis.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place",
				"threshold = 80 -> 90",
			},
			expectedNotContains: []string{
				"Terraform will perform the following actions:",
			},
		},
		{
			name:                "output without start marker",
			terraformOutput:     "Some other terraform output without the expected marker",
			expectedContains:    []string{},
			expectedNotContains: []string{},
		},
		{
			name:            "output without end marker",
			terraformOutput: "Terraform will perform the following actions:\n\n  # aws_instance.example will be updated in-place\n  ~ resource \"aws_instance\" \"example\" {\n        id = \"i-1234567890abcdef0\"\n      ~ instance_type = \"t2.micro\" -> \"t2.small\"\n        # (10 unchanged attributes hidden)\n    }",
			expectedContains: []string{
				"aws_instance.example will be updated in-place",
				"instance_type = \"t2.micro\" -> \"t2.small\"",
			},
			expectedNotContains: []string{
				"Terraform will perform the following actions:",
			},
		},
		{
			name:            "large terraform output with truncation",
			terraformOutput: "Terraform will perform the following actions:\n\n  # module.mflt-chunks-ingress-reassembly-redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionChunksIngressReassemblyRedis-001 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-chunks-ingress-reassembly-redis.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionChunksIngressReassemblyRedis-002 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-coolify-redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionCoolifyRedis-001 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-coolify-redis.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionCoolifyRedis-002 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-diagnostic-logs-redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionDiagnosticLogsRedis-001 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-diagnostic-logs-redis.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionDiagnosticLogsRedis-002 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-rate-limit-redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionRateLimitRedis-001 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-rate-limit-redis.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n        id                                    = \"ProductionRateLimitRedis-002 High CPU Alarm\"\n        tags                                  = {}\n      ~ threshold                             = 80 -> 90\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-redis-prod-primary.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n      - datapoints_to_alarm                   = 2 -> null\n        id                                    = \"ProductionRedisPrimary-001 High CPU Alarm\"\n        tags                                  = {}\n        # (21 unchanged attributes hidden)\n    }\n\n  # module.mflt-redis-prod-primary.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place\n  ~ resource \"aws_cloudwatch_metric_alarm\" \"high-cpu\" {\n      - datapoints_to_alarm                   = 2 -> null\n        id                                    = \"ProductionRedisPrimary-002 High CPU Alarm\"\n        tags                                  = {}\n        # (21 unchanged attributes hidden)\n    }\n\nPlan: 0 to add, 10 to change, 0 to destroy.\n\n",
			expectedContains: []string{
				"module.mflt-chunks-ingress-reassembly-redis.aws_cloudwatch_metric_alarm.high-cpu[0] will be updated in-place",
				"...",
				"Plan: 0 to add, 10 to change, 0 to destroy.",
			},
			expectedNotContains: []string{
				"Terraform will perform the following actions:",
				"module.mflt-redis-prod-primary.aws_cloudwatch_metric_alarm.high-cpu[1] will be updated in-place",
			},
		},
	}

	formatter := NewMemfaultSlackFormatter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.extractDriftDetails(tt.terraformOutput)

			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("extractDriftDetails() should contain '%s', but got: %s", expected, result)
				}
			}

			for _, notExpected := range tt.expectedNotContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("extractDriftDetails() should not contain '%s', but got: %s", notExpected, result)
				}
			}
		})
	}
}
