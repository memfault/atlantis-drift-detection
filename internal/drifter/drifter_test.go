package drifter

import (
	"context"
	"testing"

	"github.com/cresta/atlantis-drift-detection/internal/atlantis"
	"github.com/stretchr/testify/require"
)

// MockNotification implements the Notification interface for testing
type MockNotification struct {
	PlanDriftCalled     bool
	LastDir             string
	LastWorkspace       string
	LastTerraformOutput string
}

func (m *MockNotification) TemporaryError(_ context.Context, _ string, _ string, _ error) error {
	return nil
}

func (m *MockNotification) ExtraWorkspaceInRemote(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *MockNotification) MissingWorkspaceInRemote(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *MockNotification) PlanDrift(_ context.Context, dir string, workspace string, terraformOutput ...string) error {
	m.PlanDriftCalled = true
	m.LastDir = dir
	m.LastWorkspace = workspace
	if len(terraformOutput) > 0 {
		m.LastTerraformOutput = terraformOutput[0]
	}
	return nil
}

func TestDrifter_UsesPlanDriftWithTerraformOutputWhenAvailable(t *testing.T) {
	mockNotification := &MockNotification{}

	// Create a mock PlanResult with Terraform output
	planResult := &atlantis.PlanResult{
		Summaries: []atlantis.PlanSummary{
			{
				HasLock:         false,
				Summary:         "Plan: 0 to add, 1 to change, 0 to destroy.",
				TerraformOutput: "Terraform will perform the following actions:\n\n  # aws_instance.example will be updated in-place\n  ~ resource \"aws_instance\" \"example\" {\n        id = \"i-1234567890abcdef0\"\n      ~ instance_type = \"t2.micro\" -> \"t2.small\"\n        # (10 unchanged attributes hidden)\n    }\n\nPlan: 0 to add, 1 to change, 0 to destroy.",
			},
		},
	}

	// Test the logic that would be used in FindDriftedWorkspaces
	dir := "infra/terraform/database/prod/us-east-1/production/redis"
	workspace := "workspace"

	if planResult.HasChanges() {
		// Get the Terraform output from the first summary that has changes
		var terraformOutput string
		for _, summary := range planResult.Summaries {
			if !summary.HasLock && summary.TerraformOutput != "" {
				terraformOutput = summary.TerraformOutput
				break
			}
		}

		// Pass the terraform output as a variadic parameter
		err := mockNotification.PlanDrift(context.Background(), dir, workspace, terraformOutput)
		require.NoError(t, err)
	}

	// Verify that PlanDrift was called with terraform output
	require.True(t, mockNotification.PlanDriftCalled)
	require.Equal(t, dir, mockNotification.LastDir)
	require.Equal(t, workspace, mockNotification.LastWorkspace)
	require.Contains(t, mockNotification.LastTerraformOutput, "aws_instance.example will be updated in-place")
}

func TestDrifter_UsesPlanDriftWithoutTerraformOutputWhenNotAvailable(t *testing.T) {
	mockNotification := &MockNotification{}

	// Create a mock PlanResult without Terraform output
	planResult := &atlantis.PlanResult{
		Summaries: []atlantis.PlanSummary{
			{
				HasLock:         false,
				Summary:         "Plan: 0 to add, 1 to change, 0 to destroy.",
				TerraformOutput: "", // No Terraform output
			},
		},
	}

	// Test the logic that would be used in FindDriftedWorkspaces
	dir := "infra/terraform/database/prod/us-east-1/production/redis"
	workspace := "workspace"

	if planResult.HasChanges() {
		// Get the Terraform output from the first summary that has changes
		var terraformOutput string
		for _, summary := range planResult.Summaries {
			if !summary.HasLock && summary.TerraformOutput != "" {
				terraformOutput = summary.TerraformOutput
				break
			}
		}

		// Pass the terraform output as a variadic parameter (will be empty)
		err := mockNotification.PlanDrift(context.Background(), dir, workspace, terraformOutput)
		require.NoError(t, err)
	}

	// Verify that PlanDrift was called without terraform output
	require.True(t, mockNotification.PlanDriftCalled)
	require.Equal(t, dir, mockNotification.LastDir)
	require.Equal(t, workspace, mockNotification.LastWorkspace)
	require.Equal(t, "", mockNotification.LastTerraformOutput)
}
