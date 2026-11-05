package walstreaming_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestLateWalStoragePublication(t *testing.T) {
	t.Parallel()
	t.Run("queues late files without publishing into the replaced storage", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		oldStorage := apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId)
		team.StorageID = oldStorage.StorageID

		otherInstance := team.App.StartConcurrentInstance(t)
		otherOwner := team.Owner
		otherOwner.Client = otherInstance.NewSessionClient(t, team.Owner.Token)

		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		rotateWal(t, source)
		team.waitForWal(t, 1)
		team.App.WaitForPhysicalBackups(t)
		team.App.PausePhysicalJobs(t)

		registrationLock := team.App.BeginSQLTransaction(t)
		lockKey := "storage-registration-" + uuid.NewString()
		registrationLock.ExecuteSQL(t, "SELECT pg_advisory_xact_lock(hashtextextended(?,0))", lockKey)
		var lockOwner int
		registrationLock.QuerySQL(t, &lockOwner, "SELECT pg_backend_pid()")
		team.App.ExecuteSQL(t, fmt.Sprintf(`CREATE FUNCTION hold_wal_registration() RETURNS trigger AS $$ BEGIN
			IF NEW.storage_id='%s' AND NEW.file_name LIKE '%s-WAL-%%' AND NEW.file_name NOT LIKE '%%.metadata' AND NEW.not_before>NEW.created_at THEN
			PERFORM pg_advisory_xact_lock(hashtextextended('%s',0)); END IF; RETURN NEW; END $$ LANGUAGE plpgsql`, oldStorage.StorageID, team.Database.Id, lockKey))
		team.App.ExecuteSQL(
			t,
			`CREATE TRIGGER hold_wal_registration BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION hold_wal_registration()`,
		)

		rotateWal(t, source)
		testwait.Until(t, "the old WAL writer pauses before registering its next file", func(context.Context) error {
			var waiting int
			team.App.QuerySQL(t, &waiting, `SELECT count(*) FROM pg_locks waiting
				JOIN pg_locks held USING (locktype,database,classid,objid,objsubid)
				WHERE NOT waiting.granted AND held.granted AND held.pid=? AND waiting.locktype='advisory'`, lockOwner)
			if waiting == 0 {
				return fmt.Errorf("the next WAL file has not reached registration")
			}
			return nil
		})

		targetWorkspace := apptest.CreateWorkspace(t, otherOwner, "Late WAL files")
		target := apptest.CreateStorageNamespace(t, otherOwner, targetWorkspace.Id)
		apptest.TransferToStorage(
			t,
			otherOwner,
			apptest.StorageSwitchTransfer{
				DatabaseID:  team.Database.Id,
				WorkspaceID: targetWorkspace.Id,
				StorageID:   target.StorageID,
			},
		)

		var capturedFiles []string
		team.App.QuerySQL(
			t,
			&capturedFiles,
			"SELECT file_name FROM storages_pending_file_deletions WHERE storage_id=?",
			oldStorage.StorageID,
		)
		require.NotEmpty(t, capturedFiles)

		registrationLock.Commit(t)
		var lateFiles []string
		testwait.Until(
			t,
			"the old WAL writer creates and discards files after switch acceptance",
			func(context.Context) error {
				team.App.QuerySQL(t, &lateFiles, `SELECT pending.file_name FROM storages_pending_file_deletions pending
				WHERE pending.storage_id=? AND pending.cleanup_database_id=? AND pending.file_name LIKE ? AND pending.generation>0 AND pending.file_name NOT IN ?`, oldStorage.StorageID, team.Database.Id, team.Database.Id.String()+"-WAL-%", capturedFiles)
				if len(lateFiles) < 2 {
					return fmt.Errorf("the late WAL artifact and metadata have not been discarded yet")
				}
				return nil
			},
		)
		for _, fileName := range lateFiles {
			oldStorage.RequireFile(t, fileName)
		}

		var oldSegments int
		team.App.QuerySQL(
			t,
			&oldSegments,
			"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=?",
			team.Database.Id,
			oldStorage.StorageID,
		)
		assert.Zero(t, oldSegments)

		team.App.ResumePhysicalJobs(t)
		testwait.Until(t, "independent cleanup removes the late writer files", func(context.Context) error {
			team.App.RunJobPass(t, "storage_file_deletion")
			var pending int
			team.App.QuerySQL(
				t,
				&pending,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND file_name IN ?",
				oldStorage.StorageID,
				lateFiles,
			)
			if pending > 0 {
				return fmt.Errorf("late writer files remain pending: %d", pending)
			}
			return nil
		})
	})
	t.Run("rejects the old streamer's uploads after another instance accepts the switch", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		oldStorage := apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId)
		team.StorageID = oldStorage.StorageID
		held := oldStorage.HoldNextUpload(t, team.Owner, team.Database.Id.String()+"-WAL-")
		otherInstance := team.App.StartConcurrentInstance(t)
		otherOwner := team.Owner
		otherOwner.Client = otherInstance.NewSessionClient(t, team.Owner.Token)

		// The target comes before the hold: a held WAL upload ends after 2 minutes, and starting the
		// target's storage inside that window can outlast it on a busy machine.
		targetWorkspace := apptest.CreateWorkspace(t, otherOwner, "Concurrent WAL switch")
		target := apptest.CreateStorageNamespace(t, otherOwner, targetWorkspace.Id)

		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		rotateWal(t, source)

		var oldFileName string
		select {
		case oldFileName = <-held.Started:
		case <-t.Context().Done():
			t.Fatal("the old WAL upload did not reach storage")
		}

		team.App.WaitForPhysicalBackups(t)
		team.App.PausePhysicalJobs(t)

		apptest.TransferToStorage(t, otherOwner, apptest.StorageSwitchTransfer{
			DatabaseID: team.Database.Id, WorkspaceID: targetWorkspace.Id, StorageID: target.StorageID,
		})

		require.True(
			t,
			team.live(t).IsStreamerRunning,
			"the accepting instance cannot cancel another instance's process",
		)

		var acceptedGeneration int
		team.App.QuerySQL(
			t,
			&acceptedGeneration,
			"SELECT generation FROM storages_pending_file_deletions WHERE storage_id=? AND file_name=?",
			oldStorage.StorageID,
			oldFileName,
		)

		var captured int
		team.App.QuerySQL(
			t,
			&captured,
			"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=? AND storage_id=? AND file_name=?",
			team.Database.Id,
			oldStorage.StorageID,
			oldFileName,
		)
		require.Equal(t, 1, captured)
		otherInstance.RunJobPass(t, "storage_file_deletion")
		var capturedCount int
		team.App.QuerySQL(
			t,
			&capturedCount,
			"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=? AND storage_id=? AND file_name=?",
			team.Database.Id,
			oldStorage.StorageID,
			oldFileName,
		)
		require.Equal(t, 1, capturedCount, "a peer's active upload must remain protected from deletion")

		assert.Positive(t, apptest.GetBackupConfig(t, otherOwner, team.Database.Id).Cleanup.PendingFileCount)

		held.Release()
		select {
		case <-held.Finished:
		case <-t.Context().Done():
			t.Fatal("the held WAL upload did not finish")
		}

		testwait.Until(t, "the old upload is discarded without confirming its receipt", func(context.Context) error {
			var generation int
			team.App.QuerySQL(
				t,
				&generation,
				"SELECT generation FROM storages_pending_file_deletions WHERE storage_id=? AND file_name=?",
				oldStorage.StorageID,
				oldFileName,
			)
			if generation <= acceptedGeneration {
				return fmt.Errorf("the old upload has not handed its file back")
			}
			return nil
		})
		oldStorage.RequireFile(t, oldFileName)
		testwait.Until(
			t,
			"the accepting instance cleans the discarded upload after its writer ends",
			func(context.Context) error {
				otherInstance.RunJobPass(t, "storage_file_deletion")
				team.App.QuerySQL(
					t,
					&capturedCount,
					"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=? AND storage_id=? AND file_name=?",
					team.Database.Id,
					oldStorage.StorageID,
					oldFileName,
				)
				if capturedCount != 0 {
					return fmt.Errorf("the discarded upload remains pending")
				}
				return nil
			},
		)

		_, storageErr := oldStorage.Client.StatObject(
			t.Context(),
			oldStorage.Bucket,
			oldFileName,
			minio.StatObjectOptions{},
		)
		require.Error(t, storageErr)
		assert.Equal(t, "NoSuchKey", minio.ToErrorResponse(storageErr).Code)

		rotateWal(t, source)
		observationEnds := time.Now().Add(5 * time.Second)
		testwait.Until(
			t,
			"the uncancelled old receiver processes subsequent WAL without publishing to the old storage",
			func(context.Context) error {
				var oldSegments int
				team.App.QuerySQL(
					t,
					&oldSegments,
					"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=?",
					team.Database.Id,
					oldStorage.StorageID,
				)
				require.Zero(t, oldSegments)
				if time.Now().Before(observationEnds) {
					return fmt.Errorf("watching the old receiver after its held upload finished")
				}
				return nil
			},
		)

		require.True(t, team.live(t).IsStreamerRunning)
		team.App.ResumePhysicalJobs(t)
		rotateWal(t, source)
		var newFiles []string
		testwait.Until(t, "the selected storage receives a new full and WAL", func(context.Context) error {
			team.App.QuerySQL(
				t,
				&newFiles,
				"SELECT file_name FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=? AND file_name IS NOT NULL",
				team.Database.Id,
				target.StorageID,
			)
			hasFull := false
			for _, backup := range team.listBackups(t) {
				if backup.Type == "FULL" && backup.Status == "COMPLETED" {
					hasFull = true
				}
			}
			if len(newFiles) == 0 || !hasFull {
				return fmt.Errorf("new FULL or WAL is missing")
			}
			return nil
		})
		for _, fileName := range newFiles {
			target.RequireFile(t, fileName)
		}
	})
}
