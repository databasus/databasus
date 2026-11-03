package backups_test

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const logicalBackupRetentionJobName = "backups_logical_retention"

//nolint:tparallel // The subtests share one source server.
func TestLogicalBackupRetention(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("removes the two oldest of seven backups and deletes their files when the config keeps five",
		func(t *testing.T) {
			tm := newTeam(t)
			source := newBackupSourceDatabase(t, tm, server)
			keepingFiveSettings := newLogicalBackupSettings()
			keepingFiveSettings.Retention.Policy = apiclient.LogicalRetentionPolicyCOUNT
			keepingFiveSettings.Retention.Count = 5
			saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingFiveSettings,
				apiclient.BackupEncryptionNONE)

			backupIDs := make([]uuid.UUID, 0, 7)
			for range 7 {
				backupIDs = append(backupIDs, completeRestorableBackup(t, tm.owner, source.databaseID).Id)
				tm.app.Clock.Advance(time.Hour)
			}

			removedFileNames := []string{getBackupFileName(t, tm, backupIDs[0]), getBackupFileName(t, tm, backupIDs[1])}
			keptFileName := getBackupFileName(t, tm, backupIDs[2])

			tm.app.RunJobPass(t, logicalBackupRetentionJobName)

			assert.Equal(t, []uuid.UUID{backupIDs[6], backupIDs[5], backupIDs[4], backupIDs[3], backupIDs[2]},
				listLogicalBackupIDs(t, tm.owner, source.databaseID))

			tm.app.RunJobPass(t, storageFileDeletionJobName)

			for _, removedFileName := range removedFileNames {
				assert.False(t, tm.app.IsLocalStorageFilePresent(t, removedFileName))
				assert.False(t, tm.app.IsLocalStorageFilePresent(t, removedFileName+".metadata"))
			}

			assert.True(t, tm.app.IsLocalStorageFilePresent(t, keptFileName))
		})

	t.Run("removes the other expired backups when one removal fails", func(t *testing.T) {
		logSink := &errorLogSink{}
		testApp := apptest.New(t, apptest.WithLogSink(logSink))
		admin := testApp.CreateAdmin(t)
		tm := team{
			app:       testApp,
			admin:     admin,
			owner:     testApp.CreateNonAdminUser(t, admin),
			workspace: apiclient.Workspace{},
		}
		tm.workspace = apptest.CreateWorkspace(t, tm.owner, "Retention failure")
		source := newBackupSourceDatabase(t, tm, server)
		keepingOneSettings := newLogicalBackupSettings()
		keepingOneSettings.Retention.Policy = apiclient.LogicalRetentionPolicyCOUNT
		keepingOneSettings.Retention.Count = 1
		saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingOneSettings,
			apiclient.BackupEncryptionNONE)

		backupIDs := make([]uuid.UUID, 0, 4)
		for range 4 {
			backupIDs = append(backupIDs, completeRestorableBackup(t, tm.owner, source.databaseID).Id)
			tm.app.Clock.Advance(time.Hour)
		}

		refusedBackupID := backupIDs[2]
		tm.app.ExecuteSQL(t, fmt.Sprintf(`CREATE FUNCTION refuse_backup_removal() RETURNS trigger AS $$ BEGIN
			IF OLD.id = '%s' THEN RAISE EXCEPTION 'removal refused'; END IF;
			RETURN OLD; END $$ LANGUAGE plpgsql`, refusedBackupID))
		tm.app.ExecuteSQL(t, `CREATE TRIGGER refuse_backup_removal BEFORE DELETE ON backups_logical
			FOR EACH ROW EXECUTE FUNCTION refuse_backup_removal()`)

		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.Equal(
			t,
			[]uuid.UUID{backupIDs[3], refusedBackupID},
			listLogicalBackupIDs(t, tm.owner, source.databaseID),
		)
		assert.True(t, logSink.hasErrorNaming(refusedBackupID.String()))

		tm.app.ExecuteSQL(t, "DROP TRIGGER refuse_backup_removal ON backups_logical")
		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.Equal(t, []uuid.UUID{backupIDs[3]}, listLogicalBackupIDs(t, tm.owner, source.databaseID))
	})

	t.Run("keeps an expired backup while it runs and removes it at the first check after it ends",
		func(t *testing.T) {
			tm := newTeam(t)
			source := newBackupSourceDatabase(t, tm, server)
			keepingOneDaySettings := newLogicalBackupSettings()
			keepingOneDaySettings.Retention.Policy = apiclient.LogicalRetentionPolicyTIMEPERIOD
			keepingOneDaySettings.Retention.Period = apiclient.RetentionPeriodDAY
			saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingOneDaySettings,
				apiclient.BackupEncryptionNONE)
			runningBackup, releaseLock := startBlockedSourceBackup(t, tm, server, source)
			tm.app.Clock.Advance(25 * time.Hour)

			tm.app.RunJobPass(t, logicalBackupRetentionJobName)

			require.Equal(t, []uuid.UUID{runningBackup.Id}, listLogicalBackupIDs(t, tm.owner, source.databaseID))

			releaseLock()
			waitForNoRunningLogicalBackup(t, tm, source.databaseID)
			tm.app.RunJobPass(t, logicalBackupRetentionJobName)

			assert.Empty(t, listLogicalBackupIDs(t, tm.owner, source.databaseID))
		})

	t.Run("keeps a backup younger than one hour that the count no longer keeps", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		keepingOneSettings := newLogicalBackupSettings()
		keepingOneSettings.Retention.Policy = apiclient.LogicalRetentionPolicyCOUNT
		keepingOneSettings.Retention.Count = 1
		saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingOneSettings,
			apiclient.BackupEncryptionNONE)
		olderBackup := completeRestorableBackup(t, tm.owner, source.databaseID)
		tm.app.Clock.Advance(time.Minute)
		newerBackup := completeRestorableBackup(t, tm.owner, source.databaseID)

		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.Equal(
			t,
			[]uuid.UUID{newerBackup.Id, olderBackup.Id},
			listLogicalBackupIDs(t, tm.owner, source.databaseID),
		)
	})
}

// errorLogSink keeps the attributes of the error records the application logged.
type errorLogSink struct {
	mutex         sync.Mutex
	errorMessages []string
}

func (s *errorLogSink) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelError
}

func (s *errorLogSink) Handle(_ context.Context, record slog.Record) error {
	var message strings.Builder

	message.WriteString(record.Message)
	record.Attrs(func(attribute slog.Attr) bool {
		message.WriteString(" " + attribute.String())

		return true
	})

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.errorMessages = append(s.errorMessages, message.String())

	return nil
}

func (s *errorLogSink) WithAttrs([]slog.Attr) slog.Handler { return s }

func (s *errorLogSink) WithGroup(string) slog.Handler { return s }

func (s *errorLogSink) hasErrorNaming(text string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return slices.ContainsFunc(s.errorMessages, func(message string) bool {
		return strings.Contains(message, text)
	})
}
