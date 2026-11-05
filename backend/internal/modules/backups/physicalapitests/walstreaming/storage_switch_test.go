package walstreaming_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestTransferPhysicalWalStorage(t *testing.T) {
	t.Parallel()

	for _, isStorageCleared := range []bool{false, true} {
		scenario := "archives WAL and a fresh full in a separate namespace after a workspace transfer"
		if isStorageCleared {
			scenario = "moves retained history cleanup and starts full and WAL after transfer with cleared storage"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			team := newWalTeam(t, walTeamSpec{})
			team.enable(t)
			team.waitForReceiver(t)

			source := team.openSource(t)
			rotateWal(t, source)
			team.waitForWal(t, 1)
			team.App.WaitForPhysicalBackups(t)
			previous := team.listBackups(t)

			if isStorageCleared {
				backupConfig := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
				cleared, err := team.Owner.Client.UpdateBackupConfigWithResponse(
					t.Context(),
					team.Database.Id,
					apiclient.BackupConfigInput{
						IsBackupsEnabled: true,
						Encryption:       backupConfig.Encryption,
						Settings:         backupConfig.Settings,
					},
				)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
			}

			targetWorkspace := apptest.CreateWorkspace(t, team.Owner, "Transferred WAL")
			target := apptest.CreateStorageNamespace(t, team.Owner, targetWorkspace.Id)

			operation := apptest.TransferToStorage(t, team.Owner, apptest.StorageSwitchTransfer{
				DatabaseID: team.Database.Id, WorkspaceID: targetWorkspace.Id, StorageID: target.StorageID,
			})

			assert.Positive(t, operation.Cleanup.PendingFileCount)

			moved, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
			require.NoError(t, err)
			require.NotNil(t, moved.JSON200)
			assert.Equal(t, targetWorkspace.Id, moved.JSON200.WorkspaceId)

			var attributedFiles int
			team.App.QuerySQL(
				t,
				&attributedFiles,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=? AND storage_id=?",
				team.Database.Id,
				team.StorageID,
			)
			assert.Positive(t, attributedFiles)
			for _, old := range previous {
				for _, retained := range team.listBackups(t) {
					assert.NotEqual(t, old.ID, retained.ID)
				}
			}

			var oldSegments int
			team.App.QuerySQL(
				t,
				&oldSegments,
				"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=?",
				team.Database.Id,
				team.StorageID,
			)
			require.Zero(t, oldSegments, "old WAL history must leave the catalog at acceptance")

			rotateWal(t, source)
			var fileNames []string
			testwait.Until(t, "the new namespace receives WAL", func(context.Context) error {
				team.App.QuerySQL(
					t,
					&fileNames,
					"SELECT file_name FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=? AND file_name IS NOT NULL",
					team.Database.Id,
					target.StorageID,
				)
				hasNewFull := false
				for _, backup := range team.listBackups(t) {
					if backup.Type == "FULL" && backup.Status == "COMPLETED" {
						hasNewFull = true
						for _, old := range previous {
							assert.NotEqual(t, old.ID, backup.ID)
						}
					}
				}
				if len(fileNames) == 0 || !hasNewFull {
					return fmt.Errorf("new storage has no WAL")
				}
				return nil
			})
			for _, fileName := range fileNames {
				target.RequireFile(t, fileName)
			}
		})
	}
}

func TestReassignPhysicalWalStorage(t *testing.T) {
	t.Parallel()
	for _, isSameStorage := range []bool{false, true} {
		scenario := "starts fresh WAL and a full in another namespace after clearing storage"
		if isSameStorage {
			scenario = "starts fresh WAL and a full after reassigning the historical storage"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			team := newWalTeam(t, walTeamSpec{})
			team.enable(t)
			team.waitForReceiver(t)

			source := team.openSource(t)
			rotateWal(t, source)
			team.waitForWal(t, 1)
			team.App.WaitForPhysicalBackups(t)
			previous := team.listBackups(t)

			backupConfig := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
			cleared, err := team.Owner.Client.UpdateBackupConfigWithResponse(
				t.Context(),
				team.Database.Id,
				apiclient.BackupConfigInput{
					IsBackupsEnabled: true,
					Encryption:       backupConfig.Encryption,
					Settings:         backupConfig.Settings,
				},
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)

			targetStorageID := team.StorageID
			var target *apptest.StorageNamespace
			if !isSameStorage {
				namespace := apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId)
				target = &namespace
				targetStorageID = namespace.StorageID
			}

			assigned, err := team.Owner.Client.UpdateBackupConfigWithResponse(
				t.Context(),
				team.Database.Id,
				apiclient.BackupConfigInput{
					IsBackupsEnabled: true,
					StorageId:        &targetStorageID,
					Encryption:       backupConfig.Encryption,
					Settings:         backupConfig.Settings,
				},
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, assigned.StatusCode(), "%s", assigned.Body)

			rotateWal(t, source)
			var files []string
			testwait.Until(t, "fresh FULL and WAL appear after reassignment", func(context.Context) error {
				rows := team.listBackups(t)
				hasFull := false
				for _, backup := range rows {
					if backup.Type == "FULL" && backup.Status == "COMPLETED" {
						hasFull = true
						for _, oldBackup := range previous {
							assert.NotEqual(t, oldBackup.ID, backup.ID)
						}
					}
				}
				team.App.QuerySQL(
					t,
					&files,
					"SELECT file_name FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=? AND file_name IS NOT NULL",
					team.Database.Id,
					targetStorageID,
				)
				if !hasFull || len(files) == 0 {
					return fmt.Errorf("fresh full or WAL is missing")
				}
				return nil
			})

			if target != nil {
				for _, file := range files {
					target.RequireFile(t, file)
				}
			}
		})
	}
}
