package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/util/testing/containers"
)

const storageFileDeletionJobName = "storage_file_deletion"

func TestMoveLogicalBackupConfigToAnotherStorage(t *testing.T) {
	t.Run("removes the logical backups and deletes their files when the storage changes", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		previousStorageID := tm.createLocalStorage(t)
		nextStorageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, previousStorageID)
		completedBackup := insertLogicalBackupInStorage(t, tm, databaseID, previousStorageID, "COMPLETED")
		failedBackup := insertLogicalBackupInStorage(t, tm, databaseID, previousStorageID, "FAILED")

		savedConfig := saveLogicalBackupConfig(t, tm.owner, databaseID, nextStorageID)

		require.NotNil(t, savedConfig.StorageId)
		assert.Equal(t, nextStorageID, *savedConfig.StorageId)
		assert.Zero(t, countLogicalBackups(t, tm, databaseID))

		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, completedBackup.fileName))
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, failedBackup.fileName))
	})

	t.Run("returns 409 in_use and keeps the storage and the backups while a logical backup is in progress",
		func(t *testing.T) {
			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			previousStorageID := tm.createLocalStorage(t)
			nextStorageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, previousStorageID)
			completedBackup := insertLogicalBackupInStorage(t, tm, databaseID, previousStorageID, "COMPLETED")
			insertLogicalBackupInStorage(t, tm, databaseID, previousStorageID, "IN_PROGRESS")

			response := updateBackupConfig(t, tm.owner, databaseID,
				newLogicalBackupConfigInput(t, &nextStorageID, newLogicalBackupSettings()))

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
			requireConfigStorage(t, tm, databaseID, previousStorageID)
			assert.Equal(t, 2, countLogicalBackups(t, tm, databaseID))

			tm.app.RunJobPass(t, storageFileDeletionJobName)
			assert.True(t, tm.app.IsLocalStorageFilePresent(t, completedBackup.fileName))
		})

	t.Run("keeps the logical backups and their files when the config is saved with the same storage",
		func(t *testing.T) {
			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			storageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)
			completedBackup := insertLogicalBackupInStorage(t, tm, databaseID, storageID, "COMPLETED")

			savedConfig := saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

			require.NotNil(t, savedConfig.StorageId)
			assert.Equal(t, storageID, *savedConfig.StorageId)
			assert.Equal(t, 1, countLogicalBackups(t, tm, databaseID))

			tm.app.RunJobPass(t, storageFileDeletionJobName)
			assert.True(t, tm.app.IsLocalStorageFilePresent(t, completedBackup.fileName))
		})

	t.Run("returns 500 internal and keeps the storage and the backups when the removal fails", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		previousStorageID := tm.createLocalStorage(t)
		nextStorageID := tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, previousStorageID)
		completedBackup := insertLogicalBackupInStorage(t, tm, databaseID, previousStorageID, "COMPLETED")
		tm.app.ExecuteSQL(t, `CREATE FUNCTION refuse_logical_backup_deletion() RETURNS trigger AS $$
			BEGIN RAISE EXCEPTION 'backup removal refused by the test'; END
			$$ LANGUAGE plpgsql`)
		tm.app.ExecuteSQL(t, `CREATE TRIGGER refuse_logical_backup_deletion BEFORE DELETE ON backups_logical
			FOR EACH ROW EXECUTE FUNCTION refuse_logical_backup_deletion()`)

		response := updateBackupConfig(t, tm.owner, databaseID,
			newLogicalBackupConfigInput(t, &nextStorageID, newLogicalBackupSettings()))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusInternalServerError,
			apiclient.ProblemCodeInternal)
		assert.NotContains(t, string(response.Body), "refused by the test")
		requireConfigStorage(t, tm, databaseID, previousStorageID)
		assert.Equal(t, 1, countLogicalBackups(t, tm, databaseID))

		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, completedBackup.fileName))
	})
}

func TestMovePhysicalBackupConfigToAnotherStorage(t *testing.T) {
	physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")

	t.Run("removes the full backups kept in the previous storage after the save", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULL)
		previousStorageID := tm.createLocalStorage(t)
		nextStorageID := tm.createLocalStorage(t)
		savePhysicalBackupConfigWithStorage(t, tm, physicalDatabase.Id, previousStorageID)
		insertCompletedPhysicalFullBackup(t, tm, physicalDatabase.Id, previousStorageID)
		keptBackupID := insertCompletedPhysicalFullBackup(t, tm, physicalDatabase.Id, nextStorageID)

		savedConfig := savePhysicalBackupConfigWithStorage(t, tm, physicalDatabase.Id, nextStorageID)

		require.NotNil(t, savedConfig.StorageId)
		assert.Equal(t, nextStorageID, *savedConfig.StorageId)
		assert.Zero(t, countRows(t, tm,
			"SELECT COUNT(*) FROM physical_full_backups WHERE database_id = ? AND storage_id = ?",
			physicalDatabase.Id, previousStorageID))
		assert.Equal(t, 1, countRows(t, tm, "SELECT COUNT(*) FROM physical_full_backups WHERE id = ?",
			keptBackupID))
	})

	t.Run("keeps the full backups when the config is saved with the same storage", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULL)
		storageID := tm.createLocalStorage(t)
		savePhysicalBackupConfigWithStorage(t, tm, physicalDatabase.Id, storageID)
		keptBackupID := insertCompletedPhysicalFullBackup(t, tm, physicalDatabase.Id, storageID)

		savePhysicalBackupConfigWithStorage(t, tm, physicalDatabase.Id, storageID)

		assert.Equal(t, 1, countRows(t, tm, "SELECT COUNT(*) FROM physical_full_backups WHERE id = ?",
			keptBackupID))
	})
}

type seededLogicalBackup struct {
	id       uuid.UUID
	fileName string
}

// insertLogicalBackupInStorage writes a backup file and its record directly, because no contract
// operation creates a backup yet. Taking the file's pending deletion away is what publishing it does.
func insertLogicalBackupInStorage(
	t *testing.T,
	tm team,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	status string,
) seededLogicalBackup {
	t.Helper()

	seededBackup := seededLogicalBackup{id: uuid.New(), fileName: "backup-" + uuid.NewString()}

	tm.app.WriteUnpublishedFile(t, storageID, seededBackup.fileName)
	tm.app.ExecuteSQL(t, "DELETE FROM storages_pending_deletions WHERE storage_id = ? AND file_name = ?",
		storageID, seededBackup.fileName)
	tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, ?, ?)`, seededBackup.id, databaseID, storageID, status, seededBackup.fileName)

	return seededBackup
}

// insertCompletedPhysicalFullBackup writes the record of a finished full backup directly, because no
// contract operation creates a physical backup yet.
func insertCompletedPhysicalFullBackup(t *testing.T, tm team, databaseID, storageID uuid.UUID) uuid.UUID {
	t.Helper()

	backupID := uuid.New()
	tm.app.ExecuteSQL(t, `INSERT INTO physical_full_backups
		(id, database_id, storage_id, timeline_id, status, file_name, start_lsn, stop_lsn, completed_at)
		VALUES (?, ?, ?, 1, 'COMPLETED', ?, '0/2000028', '0/2000138', NOW())`,
		backupID, databaseID, storageID, "full-"+backupID.String())

	return backupID
}

func savePhysicalBackupConfigWithStorage(
	t *testing.T,
	tm team,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) apiclient.BackupConfig {
	t.Helper()

	readConfig := apptest.GetBackupConfig(t, tm.owner, databaseID)
	physical := getPhysicalSettings(t, readConfig)

	response := updateBackupConfig(t, tm.owner, databaseID, apiclient.BackupConfigInput{
		IsBackupsEnabled: true,
		StorageId:        &storageID,
		Encryption:       readConfig.Encryption,
		Settings:         fromPhysicalBackupSettings(t, physical),
	})
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func requireConfigStorage(t *testing.T, tm team, databaseID, expectedStorageID uuid.UUID) {
	t.Helper()

	readConfig := apptest.GetBackupConfig(t, tm.owner, databaseID)
	require.NotNil(t, readConfig.StorageId)
	require.Equal(t, expectedStorageID, *readConfig.StorageId)
}

func countLogicalBackups(t *testing.T, tm team, databaseID uuid.UUID) int {
	t.Helper()

	return countRows(t, tm, "SELECT COUNT(*) FROM backups_logical WHERE database_id = ?", databaseID)
}
