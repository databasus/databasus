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
	t.Run("keeps files first registered after acceptance outside the captured cleanup set", func(t *testing.T) {
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
			`CREATE TRIGGER hold_wal_registration BEFORE INSERT ON storages_pending_deletions FOR EACH ROW EXECUTE FUNCTION hold_wal_registration()`,
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
		operation := apptest.TransferToStorage(
			t,
			otherOwner,
			apptest.StorageSwitchTransfer{
				DatabaseID:  team.Database.Id,
				WorkspaceID: targetWorkspace.Id,
				StorageID:   target.StorageID,
			},
		)

		registrationLock.Commit(t)
		var lateFiles []string
		testwait.Until(
			t,
			"the old WAL writer creates and discards files after switch acceptance",
			func(context.Context) error {
				team.App.QuerySQL(t, &lateFiles, `SELECT pending.file_name FROM storages_pending_deletions pending
				WHERE pending.storage_id=? AND pending.is_ordinary AND pending.generation>0 AND pending.updated_at>?
				AND NOT EXISTS (SELECT 1 FROM storages_deletion_batch_items item WHERE item.batch_id=(SELECT deletion_batch_id FROM backups_storage_switches WHERE id=?) AND item.storage_id=pending.storage_id AND item.file_name=pending.file_name)`, oldStorage.StorageID, operation.CreatedAt, operation.Id)
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

		outcome := apptest.GetStorageSwitch(t, team.Owner, operation.Id)
		assert.Equal(t, operation.Total, outcome.Total)
		assert.Equal(t, operation.CleanupDeadline, outcome.CleanupDeadline)

		team.App.ResumePhysicalJobs(t)
		testwait.Until(t, "independent cleanup removes the late writer files", func(context.Context) error {
			team.App.RunJobPass(t, "storage_file_deletion")
			var pending int
			team.App.QuerySQL(
				t,
				&pending,
				"SELECT count(*) FROM storages_pending_deletions WHERE storage_id=? AND file_name IN ?",
				oldStorage.StorageID,
				lateFiles,
			)
			if pending > 0 {
				return fmt.Errorf("late writer files remain pending: %d", pending)
			}
			return nil
		})

		outcome = apptest.GetStorageSwitch(t, team.Owner, operation.Id)
		assert.Equal(t, operation.Total, outcome.Total)
		assert.Equal(t, operation.CleanupDeadline, outcome.CleanupDeadline)
	})
	t.Run("rejects the old streamer's uploads after another instance accepts the switch", func(t *testing.T) {
		team := newWalTeam(t, walTeamSpec{})
		oldStorage := apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId)
		team.StorageID = oldStorage.StorageID
		held := oldStorage.HoldNextUpload(t, team.Owner, team.Database.Id.String()+"-WAL-")
		otherInstance := team.App.StartConcurrentInstance(t)
		otherOwner := team.Owner
		otherOwner.Client = otherInstance.NewSessionClient(t, team.Owner.Token)

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

		targetWorkspace := apptest.CreateWorkspace(t, otherOwner, "Concurrent WAL switch")
		target := apptest.CreateStorageNamespace(t, otherOwner, targetWorkspace.Id)
		operation := apptest.TransferToStorage(t, otherOwner, apptest.StorageSwitchTransfer{
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
			"SELECT generation FROM storages_pending_deletions WHERE storage_id=? AND file_name=?",
			oldStorage.StorageID,
			oldFileName,
		)

		var captured int
		team.App.QuerySQL(
			t,
			&captured,
			"SELECT count(*) FROM storages_deletion_batch_items WHERE batch_id=(SELECT deletion_batch_id FROM backups_storage_switches WHERE id=?) AND storage_id=? AND file_name=?",
			operation.Id,
			oldStorage.StorageID,
			oldFileName,
		)
		require.Equal(t, 1, captured)
		otherInstance.RunJobPass(t, "storage_file_deletion")
		var capturedStatus string
		team.App.QuerySQL(
			t,
			&capturedStatus,
			"SELECT status FROM storages_deletion_batch_items WHERE batch_id=(SELECT deletion_batch_id FROM backups_storage_switches WHERE id=?) AND storage_id=? AND file_name=?",
			operation.Id,
			oldStorage.StorageID,
			oldFileName,
		)
		require.Equal(t, "PENDING", capturedStatus, "a peer's active upload must remain protected from deletion")

		outcomeWhileWriting := apptest.GetStorageSwitch(t, otherOwner, operation.Id)
		assert.Equal(t, operation.Total, outcomeWhileWriting.Total)
		assert.Positive(t, outcomeWhileWriting.Pending)

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
				"SELECT generation FROM storages_pending_deletions WHERE storage_id=? AND file_name=?",
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
					&capturedStatus,
					"SELECT status FROM storages_deletion_batch_items WHERE batch_id=(SELECT deletion_batch_id FROM backups_storage_switches WHERE id=?) AND storage_id=? AND file_name=?",
					operation.Id,
					oldStorage.StorageID,
					oldFileName,
				)
				if capturedStatus != "DELETED" {
					return fmt.Errorf("the discarded upload remains %s", capturedStatus)
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

		outcome := apptest.GetStorageSwitch(t, team.Owner, operation.Id)
		assert.Equal(t, operation.Total, outcome.Total)
		assert.Equal(t, operation.CleanupDeadline, outcome.CleanupDeadline)
	})
}
