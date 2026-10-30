package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestPhysicalBackupRunOverlap(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"cancel", "delete", "disable"} {
		t.Run("waits for the old process before completing the next full after "+action, func(t *testing.T) {
			team := newBackupTeam(t, backupTeamSpec{})
			backup, releaseStorage := startHeldPhysicalStream(t, team, "full")
			resume := suspendBackupProcess(t, backup.ID)
			releaseStorage()
			var attemptFiles []string
			team.App.QuerySQL(t, &attemptFiles, "SELECT file_name FROM backups_physical WHERE id=?", backup.ID)
			require.Len(t, attemptFiles, 1)
			require.NotEmpty(t, attemptFiles[0])
			switch action {
			case "cancel":
				cancelPhysicalBackup(t, team, backup)
				assert.Equal(t, "CANCELED", findPhysicalBackup(t, team, backup).Status)
				team.trigger(t, "full")
			case "delete":
				status, body := team.App.CallPhysicalAPI(t, team.Owner, apptest.PhysicalDeleteRequest(backup.ID))
				require.Equal(t, http.StatusNoContent, status, "%s", body)
				require.Empty(t, team.listBackups(t))
			case "disable":
				setBackupsEnabled(t, team, false)
				assert.Equal(t, "CANCELED", findPhysicalBackup(t, team, backup).Status)
				team.App.Clock.Advance(48 * time.Hour)
				setBackupsEnabled(t, team, true)
			}
			beforePass := team.listBackups(t)
			team.App.RunJobPass(t, "backups_physical_scheduler")
			require.Equal(
				t,
				beforePass,
				team.listBackups(t),
				"the scheduler must insert nothing while the stopped process is alive",
			)
			resume()
			var nextBackup backupRow
			testwait.Until(
				t,
				"the next scheduler pass completes a full after the old run ends",
				func(context.Context) error {
					team.App.RunJobPass(t, "backups_physical_scheduler")
					for _, listed := range team.listBackups(t) {
						if listed.ID != backup.ID {
							if listed.Status == "IN_PROGRESS" {
								return fmt.Errorf("the next full is still running")
							}
							require.Equal(t, "COMPLETED", listed.Status)
							nextBackup = listed
							return nil
						}
					}
					return fmt.Errorf("the old run has not ended yet")
				},
			)
			assert.Equal(t, "FULL", nextBackup.Type)
			if action != "delete" {
				assert.Equal(t, "CANCELED", findPhysicalBackup(t, team, backup).Status)
			}
			var pendingFiles []string
			team.App.QuerySQL(
				t,
				&pendingFiles,
				"SELECT file_name FROM storages_pending_deletions WHERE storage_id=?",
				team.StorageID,
			)
			for _, suffix := range []string{"", ".metadata", ".manifest"} {
				assert.Contains(t, pendingFiles, attemptFiles[0]+suffix)
			}
		})
	}
}

func setBackupsEnabled(t *testing.T, team *backupTeam, isEnabled bool) {
	t.Helper()
	config := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
	response, err := team.Owner.Client.UpdateBackupConfigWithResponse(t.Context(), team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: isEnabled, StorageId: config.StorageId,
			Encryption: config.Encryption, Settings: config.Settings,
		})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}

func suspendBackupProcess(t *testing.T, backupID uuid.UUID) func() {
	t.Helper()
	processes, err := os.ReadDir("/proc")
	require.NoError(t, err)
	for _, entry := range processes {
		processID, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		arguments, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || !strings.Contains(string(arguments), "pg_basebackup") ||
			!strings.Contains(string(arguments), backupID.String()) {
			continue
		}
		require.NoError(t, syscall.Kill(processID, syscall.SIGSTOP))
		resume := func() {
			currentArguments, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
			if err == nil && string(currentArguments) == string(arguments) {
				_ = syscall.Kill(processID, syscall.SIGCONT)
			}
		}
		t.Cleanup(resume)
		return resume
	}
	t.Fatal("the backup's pg_basebackup process was not found")
	return nil
}
