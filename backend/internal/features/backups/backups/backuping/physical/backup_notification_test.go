package backuping_physical

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
)

func TestClassifyBackupNotification(t *testing.T) {
	for _, backupKind := range []string{"full", "incremental"} {
		t.Run(backupKind, func(t *testing.T) {
			for _, testCase := range []struct {
				name            string
				status          physical_enums.PhysicalBackupStatus
				expectedTrigger backups.NotificationTrigger
				headingFragment string
			}{
				{"returns a success notification when the backup completes", physical_enums.PhysicalBackupStatusCompleted, backups.NotificationTriggerBackupSuccess, "completed"},
				{"returns a failure notification when the backup fails", physical_enums.PhysicalBackupStatusError, backups.NotificationTriggerBackupFailed, "failed"},
				{"returns a chain notification when the chain breaks", physical_enums.PhysicalBackupStatusChainBroken, backups.NotificationTriggerChainBroken, "chain-broken"},
				{"returns no notification when the backup is in progress", physical_enums.PhysicalBackupStatusInProgress, "", ""},
				{"returns no notification when the backup is canceled", physical_enums.PhysicalBackupStatusCanceled, "", ""},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					backupID := uuid.New()
					spec := backupNotificationSpec{
						Database:      databases.Database{Name: "orders"},
						WorkspaceName: "Production",
						BackupResult: postgresql_executor.PhysicalBackupResult{
							BackupSizeMb:     12.5,
							BackupDurationMs: 999,
							ErrorMessage:     "source unavailable",
						},
						FullBackup: &physical_models.PhysicalFullBackup{
							ID:          backupID,
							Status:      testCase.status,
							ErrorReason: new(physical_enums.PhysicalBackupErrorPgBasebackupFailed),
						},
						IncrementalBackup: &physical_models.PhysicalIncrementalBackup{
							ID:          backupID,
							Status:      testCase.status,
							ErrorReason: new(physical_enums.PhysicalBackupErrorPgBasebackupFailed),
						},
					}
					classify := classifyFullNotification
					if backupKind == "incremental" {
						classify = classifyIncrNotification
					}

					trigger, heading, message := classify(spec)

					assert.Equal(t, testCase.expectedTrigger, trigger)
					if testCase.expectedTrigger == "" {
						assert.Empty(t, heading)
						assert.Empty(t, message)
					} else {
						assert.Contains(t, heading, testCase.headingFragment)
						assert.Contains(t, heading, "orders")
						assert.Contains(t, heading, "Production")
						assert.Contains(t, message, backupID.String())
					}
				})
			}
		})
	}
}
