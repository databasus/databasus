package backuping

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
)

func TestClassifyBackupNotification(t *testing.T) {
	for _, backupKind := range []string{"full", "incremental"} {
		t.Run(backupKind, func(t *testing.T) {
			for _, testCase := range []struct {
				name            string
				status          catalog.BackupStatus
				expectedTrigger config.NotificationTrigger
				headingFragment string
			}{
				{"returns a success notification when the backup completes", catalog.BackupStatusCompleted, config.NotificationTriggerBackupSuccess, "completed"},
				{"returns a failure notification when the backup fails", catalog.BackupStatusError, config.NotificationTriggerBackupFailed, "failed"},
				{"returns a chain notification when the chain breaks", catalog.BackupStatusChainBroken, config.NotificationTriggerChainBroken, "chain-broken"},
				{"returns no notification when the backup is in progress", catalog.BackupStatusInProgress, "", ""},
				{"returns no notification when the backup is canceled", catalog.BackupStatusCanceled, "", ""},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					backupID := uuid.New()
					spec := backupNotificationSpec{
						Database:      databases.Database{Name: "orders"},
						WorkspaceName: "Production",
						BackupResult: Result{
							BackupSizeMb:     12.5,
							BackupDurationMs: 999,
							ErrorMessage:     "source unavailable",
						},
						Backup: &catalog.Backup{
							Kind: catalog.BackupKindFull, ID: backupID, Status: testCase.status,
							ErrorReason: new(catalog.BackupErrorPgBasebackupFailed),
						},
					}
					if backupKind == "incremental" {
						spec.Backup.Kind = catalog.BackupKindIncremental
					}
					trigger, heading, message := classifyBackupNotificationSpec(spec)

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
