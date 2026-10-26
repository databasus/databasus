package usecases_logical_mongodb

import (
	"context"
	"testing"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	databases_bridge "databasus-backend/internal/features/databases"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/testing/containers"
)

// Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816 pins issue #816 for
// MongoDB: a storage timeout that ended the backup context was reported as "backup cancelled".
func Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816(t *testing.T) {
	endpoint := containers.StartMongodb(t, "mongo:7.0")
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)
	database := databases_bridge.CreateTestMongodbDatabase(endpoint.Host, endpoint.Port, workspace.ID, nil)

	t.Cleanup(func() {
		databases_bridge.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	})

	backups_core_logical.AssertStorageTimeoutReportedAsStorageFailure(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
	) error {
		_, err := GetCreateMongodbBackupUsecase().Execute(ctx, backup, backupConfig, database, fileStore, func(float64) {})

		return err
	})
}

func Test_Execute_WhenMongodumpFailsBeforeProducingData_ReportsMongodumpError(t *testing.T) {
	backups_core_logical.AssertDumpToolErrorReported(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
		dumpToolBin string,
	) error {
		_, err := GetCreateMongodbBackupUsecase().streamToStorage(
			ctx, backup, backupConfig, dumpToolBin, nil, fileStore, func(float64) {},
		)

		return err
	})
}
