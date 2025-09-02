package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func genericNotificationTest(t *testing.T, notification Notification) {
	ctx := context.Background()
	require.NoError(t, notification.ExtraWorkspaceInRemote(ctx, "genericNotificationTest/ExtraWorkspaceInRemote", "test-workspace"))
	require.NoError(t, notification.MissingWorkspaceInRemote(ctx, "genericNotificationTest/MissingWorkspaceInRemote", "test-workspace"))
	require.NoError(t, notification.PlanDrift(ctx, "infra/terraform/database/prod/us-east-1/demo/lavinmq", "infra_terraform_database_prod_us-east-1_demo_lavinmq"))
}
