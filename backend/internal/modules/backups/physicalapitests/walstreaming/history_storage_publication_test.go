package walstreaming_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestHistoryReuseStorageGeneration(t *testing.T) {
	t.Parallel()

	t.Run("rejects reuse by an old generation when the selected storage is switched away and back", func(t *testing.T) {
		t.Parallel()

		primary, standby := containers.StartPhysicalPrimaryWithStandby(t, "postgres:17")
		barrier := holdNextHistoryAcquisition(t, standby)
		observer := historyObserver{Handler: slog.DiscardHandler, events: make(chan string, 1024)}
		team := newWalTeam(t, walTeamSpec{SourceEndpoint: &barrier.endpoint, LogSink: observer})
		t.Cleanup(barrier.Release)
		team.Source = standby
		oldStorage := apptest.CreateStorageNamespace(t, team.Owner, team.Workspace.Id)
		target := apptest.CreateStorageNamespace(t, team.Owner, team.Workspace.Id)
		team.StorageID = oldStorage.StorageID
		otherInstance := team.App.StartConcurrentInstance(t)
		otherOwner := team.Owner
		otherOwner.Client = otherInstance.NewSessionClient(t, team.Owner.Token)

		team.enable(t)
		team.waitForReceiver(t)
		primaryTeam := *team
		primaryTeam.Source = primary
		rotateWal(t, primaryTeam.openSource(t))
		team.waitForWal(t, 1)
		team.App.WaitForPhysicalBackups(t)
		team.App.PausePhysicalJobs(t)

		source := team.openSource(t)
		var isPromoted bool
		require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
		require.True(t, isPromoted)

		historyPath := team.App.GetWalQueueFilePath(team.Database.Id, "00000002.history")
		require.NoError(t, os.WriteFile(historyPath, []byte("notification only\n"), 0o600))
		testwait.Until(t, "the old history acquisition pauses before the catalog lookup", func(context.Context) error {
			select {
			case <-barrier.started:
				return nil
			default:
				return fmt.Errorf("history acquisition has not reached the barrier")
			}
		})

		apptest.SaveBackupConfigWithStorage(t, otherOwner, apptest.StorageUse{
			DatabaseID: team.Database.Id, StorageID: target.StorageID,
		})
		apptest.SaveBackupConfigWithStorage(t, otherOwner, apptest.StorageUse{
			DatabaseID: team.Database.Id, StorageID: oldStorage.StorageID,
		})
		status, body := otherInstance.CallPhysicalAPI(
			t,
			otherOwner,
			apptest.PhysicalStartRequest(team.Database.Id, "FULL"),
		)
		require.Equal(t, http.StatusAccepted, status, "%s", body)
		otherInstance.RunJobPass(t, "backups_physical_scheduler")

		testwait.Until(
			t,
			"the current generation publishes history while the old acquisition remains held",
			func(context.Context) error {
				histories := team.getPublishedHistory(t)
				if len(histories) != 1 || histories[0].StorageID != oldStorage.StorageID {
					return fmt.Errorf("the current generation has not published history")
				}

				return nil
			},
		)
		otherInstance.WaitForPhysicalBackups(t)
		histories := team.getPublishedHistory(t)
		require.Len(t, histories, 1)
		oldStorage.RequireFile(t, histories[0].FileName)
		oldStorage.RequireFile(t, histories[0].FileName+".metadata")

		barrier.Release()

		testwait.Until(
			t,
			"the old generation refuses the published history instead of acknowledging it",
			func(context.Context) error {
				select {
				case event := <-observer.events:
					require.Equal(t, "history upload failed; will retry next tick", event,
						"an old generation must not acknowledge the current generation's history")

					return nil
				default:
					return fmt.Errorf("the old history attempt has not finished")
				}
			},
		)
		require.FileExists(t, historyPath)
		require.Equal(t, histories, team.getPublishedHistory(t))
		oldStorage.RequireFile(t, histories[0].FileName)
		oldStorage.RequireFile(t, histories[0].FileName+".metadata")
	})
}

func TestLateHistoryStoragePublication(t *testing.T) {
	t.Parallel()

	t.Run(
		"attributes history artifacts and sidecars to outstanding cleanup when registration follows a peer storage switch",
		func(t *testing.T) {
			t.Parallel()

			primary, standby := containers.StartPhysicalPrimaryWithStandby(t, "postgres:17")
			team := newWalTeam(t, walTeamSpec{SourceEndpoint: &standby})
			oldStorage := apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId)
			target := apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId)
			team.StorageID = oldStorage.StorageID
			otherInstance := team.App.StartConcurrentInstance(t)
			otherOwner := team.Owner
			otherOwner.Client = otherInstance.NewSessionClient(t, team.Owner.Token)
			team.enable(t)
			team.waitForReceiver(t)
			primaryTeam := *team
			primaryTeam.Source = primary
			primarySource := primaryTeam.openSource(t)
			rotateWal(t, primarySource)
			team.waitForWal(t, 1)
			team.App.WaitForPhysicalBackups(t)
			team.App.PausePhysicalJobs(t)

			registrationLock := team.App.BeginSQLTransaction(t)
			lockKey := "history-registration-" + uuid.NewString()
			registrationLock.ExecuteSQL(t, "SELECT pg_advisory_xact_lock(hashtextextended(?,0))", lockKey)
			var lockOwner int
			registrationLock.QuerySQL(t, &lockOwner, "SELECT pg_backend_pid()")
			team.App.ExecuteSQL(t, fmt.Sprintf(`CREATE FUNCTION hold_history_registration() RETURNS trigger AS $$ BEGIN
   IF NEW.storage_id='%s' AND NEW.file_name LIKE '%s-HIST-%%' AND NEW.file_name NOT LIKE '%%.metadata' AND NEW.not_before>NEW.created_at THEN
    PERFORM pg_advisory_xact_lock(hashtextextended('%s',0));
   END IF; RETURN NEW; END $$ LANGUAGE plpgsql`, oldStorage.StorageID, team.Database.Id, lockKey))
			team.App.ExecuteSQL(
				t,
				`CREATE TRIGGER hold_history_registration BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION hold_history_registration()`,
			)
			source := team.openSource(t)
			var isPromoted bool
			require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
			require.True(t, isPromoted)
			rotateWal(t, source)
			testwait.Until(t, "the history upload pauses before registering its artifact", func(context.Context) error {
				var waiting int
				team.App.QuerySQL(t, &waiting, `SELECT count(*) FROM pg_locks waiting
    JOIN pg_locks held USING (locktype,database,classid,objid,objsubid)
    WHERE NOT waiting.granted AND held.granted AND held.pid=? AND waiting.locktype='advisory'`, lockOwner)

				if waiting == 0 {
					return fmt.Errorf("history registration has not reached the barrier")
				}

				return nil
			})

			apptest.SaveBackupConfigWithStorage(
				t,
				otherOwner,
				apptest.StorageUse{DatabaseID: team.Database.Id, StorageID: target.StorageID},
			)
			accepted := apptest.GetBackupConfig(t, otherOwner, team.Database.Id)
			require.Positive(t, accepted.Cleanup.PendingFileCount)
			var capturedHistory int
			team.App.QuerySQL(
				t,
				&capturedHistory,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND file_name LIKE ?",
				oldStorage.StorageID,
				team.Database.Id.String()+"-HIST-%",
			)
			require.Zero(t, capturedHistory)
			registrationLock.Commit(t)

			var historyFiles []string
			testwait.Until(
				t,
				"the late history artifact and sidecar inherit the original cleanup scope",
				func(context.Context) error {
					team.App.QuerySQL(t, &historyFiles, `SELECT file_name FROM storages_pending_file_deletions
    WHERE storage_id=? AND cleanup_database_id=? AND cleanup_workspace_id=? AND file_name LIKE ? AND generation>0`, oldStorage.StorageID, team.Database.Id, team.Workspace.Id, team.Database.Id.String()+"-HIST-%")

					for _, fileName := range historyFiles {
						if strings.HasSuffix(fileName, ".metadata") &&
							slices.Contains(historyFiles, strings.TrimSuffix(fileName, ".metadata")) {
							return nil
						}
					}

					return fmt.Errorf(
						"no attributed history artifact and metadata pair among %d files",
						len(historyFiles),
					)
				},
			)
			for _, fileName := range historyFiles {
				oldStorage.RequireFile(t, fileName)
			}
			team.StorageID = target.StorageID
			team.disable(t)

			testwait.Until(
				t,
				"the old streamer finishes its discarded uploads after disabling backups",
				func(context.Context) error {
					if team.live(t).IsStreamerRunning {
						return fmt.Errorf("the old streamer is still stopping")
					}
					return nil
				},
			)

			var scopedFiles int64
			team.App.QuerySQL(
				t,
				&scopedFiles,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=? AND cleanup_workspace_id=?",
				team.Database.Id,
				team.Workspace.Id,
			)
			assert.Equal(
				t,
				scopedFiles,
				apptest.GetBackupConfig(t, otherOwner, team.Database.Id).Cleanup.PendingFileCount,
			)
			assert.GreaterOrEqual(t, scopedFiles, int64(len(historyFiles)))
			var unscopedHistory int
			team.App.QuerySQL(
				t,
				&unscopedHistory,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND file_name LIKE ? AND cleanup_database_id IS NULL",
				oldStorage.StorageID,
				team.Database.Id.String()+"-HIST-%",
			)
			assert.Zero(t, unscopedHistory)

			var publishedHistory int
			team.App.QuerySQL(
				t,
				&publishedHistory,
				"SELECT count(*) FROM backups_physical_wal_history_files WHERE database_id=? AND storage_id=?",
				team.Database.Id,
				oldStorage.StorageID,
			)
			assert.Zero(t, publishedHistory)
			testwait.Until(t, "cleanup confirms the absence of every old file", func(context.Context) error {
				otherInstance.RunJobPass(t, "storage_file_deletion")

				pending := apptest.GetBackupConfig(t, otherOwner, team.Database.Id).Cleanup.PendingFileCount
				if pending > 0 {
					return fmt.Errorf("old files still pending: %d", pending)
				}

				return nil
			})
			for _, fileName := range historyFiles {
				oldStorage.RequireFileAbsent(t, fileName)
			}
			assert.Zero(t, apptest.GetBackupConfig(t, otherOwner, team.Database.Id).Cleanup.PendingFileCount)

			team.enable(t)
			team.App.ResumePhysicalJobs(t)
			testwait.Until(t, "the current storage generation publishes its own history", func(context.Context) error {
				histories := team.getPublishedHistory(t)
				if len(histories) != 1 || histories[0].StorageID != target.StorageID {
					return fmt.Errorf("no history in the current storage")
				}

				target.RequireFile(t, histories[0].FileName)
				target.RequireFile(t, histories[0].FileName+".metadata")

				return nil
			})
		},
	)
}
