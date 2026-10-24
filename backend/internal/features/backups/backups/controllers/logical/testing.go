//go:build testhelpers

package backups_controllers_logical

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

func CreateTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config_logical.GetBackupConfigController(),
		GetBackupController(),
	)

	// Register public routes (no auth required - token-based)
	v1 := router.Group("/api/v1")
	GetBackupController().RegisterPublicRoutes(v1)

	return router
}

// WaitForBackupCompletion waits for a new backup to be created and completed (or failed)
// for the given database. It checks for backups with count greater than expectedInitialCount.
func WaitForBackupCompletion(
	t *testing.T,
	databaseID uuid.UUID,
	expectedInitialCount int,
	timeout time.Duration,
) {
	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		backups, err := backups_core_logical.GetBackupRepository().FindByDatabaseID(databaseID)
		if err != nil {
			t.Logf("WaitForBackupCompletion: error finding backups: %v", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}

		t.Logf(
			"WaitForBackupCompletion: found %d backups (expected > %d)",
			len(backups),
			expectedInitialCount,
		)

		if len(backups) > expectedInitialCount {
			// Check if the newest backup has completed or failed
			newestBackup := backups[0]
			t.Logf("WaitForBackupCompletion: newest backup status: %s", newestBackup.Status)

			if newestBackup.Status == backups_core_logical.BackupStatusCompleted ||
				newestBackup.Status == backups_core_logical.BackupStatusFailed ||
				newestBackup.Status == backups_core_logical.BackupStatusCanceled {
				t.Logf(
					"WaitForBackupCompletion: backup finished with status %s",
					newestBackup.Status,
				)
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("WaitForBackupCompletion: timeout waiting for backup to complete")
}

// CreateTestBackup creates a simple test backup record for testing purposes
func CreateTestBackup(databaseID, storageID uuid.UUID) *backups_core_logical.LogicalBackup {
	backup := &backups_core_logical.LogicalBackup{
		ID:               uuid.New(),
		DatabaseID:       databaseID,
		StorageID:        storageID,
		Status:           backups_core_logical.BackupStatusCompleted,
		BackupSizeMb:     10.5,
		BackupDurationMs: 1000,
		CreatedAt:        time.Now().UTC(),
	}

	repo := &backups_core_logical.BackupRepository{}
	if err := repo.Save(backup); err != nil {
		panic(err)
	}

	return backup
}

type TestBackupOptions struct {
	Status    backups_core_logical.BackupStatus
	CreatedAt time.Time
}

func CreateTestBackupWithOptions(
	databaseID, storageID uuid.UUID,
	opts TestBackupOptions,
) *backups_core_logical.LogicalBackup {
	backup := &backups_core_logical.LogicalBackup{
		ID:               uuid.New(),
		DatabaseID:       databaseID,
		StorageID:        storageID,
		Status:           opts.Status,
		BackupSizeMb:     10.5,
		BackupDurationMs: 1000,
		CreatedAt:        opts.CreatedAt,
	}

	repo := &backups_core_logical.BackupRepository{}
	if err := repo.Save(backup); err != nil {
		panic(err)
	}

	return backup
}
