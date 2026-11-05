package databases_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages"
)

const storageFileDeletionJob = "storage_file_deletion"

//nolint:tparallel // The subtests share one source server.
func TestDeleteDatabase(t *testing.T) {
	t.Parallel()

	postgresServer := containers.StartPostgres(t, "postgres:16")

	t.Run("removes local backup files within five seconds when a database is deleted", func(t *testing.T) {
		tm := newTeam(t, apptest.WithStorageFileTimings(storages.ProductionFileTimings()))
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		storedBackup := insertPublishedLogicalBackup(t, tm, logicalBackupSpec{
			databaseID: savedDatabase.Id,
			storageID:  storageID,
			status:     "COMPLETED",
		})
		backupFiles := []string{storedBackup.fileName, storedBackup.fileName + ".metadata"}
		tm.app.PublishStorageFile(t, storages.StoredFileReference{
			StorageID: storageID,
			FileName:  backupFiles[1],
		}, strings.NewReader("backup metadata"))
		runner := tm.app.StartJobRunner(t, []string{storageFileDeletionJob})
		testwait.Until(t, "initial file cleanup pass completes", func(context.Context) error {
			for _, snapshot := range runner.GetSnapshots() {
				if snapshot.LastCompletedAt != nil && !snapshot.IsRunning {
					return nil
				}
			}

			return fmt.Errorf("file cleanup has not completed its initial pass")
		})

		deletionStarted := time.Now()
		response := deleteDatabase(t, tm.owner, savedDatabase.Id)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))

		testwait.Within(5*time.Second).
			Until(t, "local backup files are removed without the periodic tick", func(context.Context) error {
				for _, fileName := range backupFiles {
					if tm.app.IsLocalStorageFilePresent(t, fileName) {
						return fmt.Errorf("backup file %s still exists", fileName)
					}
				}

				if tm.countRows(
					t,
					"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id = ?",
					storageID,
				) != 0 {
					return fmt.Errorf("file cleanup still has pending deletions")
				}

				return nil
			})
		t.Logf("local backup files removed in %s", time.Since(deletionStarted))

		removedStorage, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), storageID, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, removedStorage.StatusCode(), string(removedStorage.Body))
	})

	t.Run("deletes the database with its backups and removes their files", func(t *testing.T) {
		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		storedBackup := insertPublishedLogicalBackup(t, tm, logicalBackupSpec{
			databaseID: savedDatabase.Id,
			storageID:  storageID,
			status:     "COMPLETED",
		})

		response := deleteDatabase(t, tm.owner, savedDatabase.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, tm.listDatabaseIDs(t))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM backups_logical WHERE database_id = ?", savedDatabase.Id))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?",
			savedDatabase.Id))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM databases_postgresql_logical WHERE database_id = ?",
			savedDatabase.Id))

		tm.app.RunJobPass(t, storageFileDeletionJob)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, storedBackup.fileName))
	})

	t.Run("returns 409 in_use and deletes nothing while a logical backup is in progress", func(t *testing.T) {
		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		runningBackup := insertPublishedLogicalBackup(t, tm, logicalBackupSpec{
			databaseID: savedDatabase.Id,
			storageID:  storageID,
			status:     "IN_PROGRESS",
		})

		response := deleteDatabase(t, tm.owner, savedDatabase.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
		assert.Equal(t, 1, tm.countRows(t, "SELECT COUNT(*) FROM backups_logical WHERE id = ?", runningBackup.id))
	})

	t.Run("keeps the database, its backups and their files when deleting the backup records fails", func(t *testing.T) {
		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		storedBackup := insertPublishedLogicalBackup(t, tm, logicalBackupSpec{
			databaseID: savedDatabase.Id,
			storageID:  storageID,
			status:     "COMPLETED",
		})
		refuseLogicalBackupDeletion(t, tm.app)

		response := deleteDatabase(t, tm.owner, savedDatabase.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusInternalServerError,
			apiclient.ProblemCodeInternal)
		assert.NotContains(t, string(response.Body), refusedBackupDeletionMessage)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
		assert.Equal(t, 1, tm.countRows(t, "SELECT COUNT(*) FROM backups_logical WHERE id = ?", storedBackup.id))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM storages_pending_file_deletions WHERE storage_id = ?",
			storageID))

		tm.app.RunJobPass(t, storageFileDeletionJob)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, storedBackup.fileName))
	})
}

//nolint:tparallel // The subtests share one source server.
func TestDeleteDatabaseAccess(t *testing.T) {
	t.Parallel()

	postgresServer := containers.StartPostgres(t, "postgres:16")

	t.Run("lets a workspace member delete a database", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response := deleteDatabase(t, member, savedDatabase.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, tm.listDatabaseIDs(t))
	})

	t.Run("returns 403 permission_denied and keeps the database when a viewer deletes it", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response := deleteDatabase(t, viewer, savedDatabase.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
	})

	t.Run("returns 404 not_found and keeps the database when an outsider deletes another team's database",
		func(t *testing.T) {
			tm := newTeam(t)
			outsider := tm.signUpNonMember(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response := deleteDatabase(t, outsider, savedDatabase.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
			assert.Equal(t, savedDatabase.Name, tm.readDatabase(t, savedDatabase.Id).Name)
		})

	t.Run("returns 401 unauthenticated and keeps the database without a session", func(t *testing.T) {
		tm := newTeam(t)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response, err := tm.app.NewClient(t).DeleteDatabaseWithResponse(t.Context(), savedDatabase.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
	})
}

func deleteDatabase(t *testing.T, session apptest.Session, databaseID uuid.UUID) *apiclient.DeleteDatabaseResponse {
	t.Helper()

	response, err := session.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)

	return response
}

type logicalBackupSpec struct {
	databaseID uuid.UUID
	storageID  uuid.UUID
	status     string
}

type storedLogicalBackup struct {
	id       uuid.UUID
	fileName string
}

// insertPublishedLogicalBackup writes a backup file and its record directly, because no contract
// operation creates a backup yet. Taking the file's pending deletion away is what publishing it does.
func insertPublishedLogicalBackup(t *testing.T, tm team, spec logicalBackupSpec) storedLogicalBackup {
	t.Helper()

	storedBackup := storedLogicalBackup{id: uuid.New(), fileName: "backup-" + uuid.NewString()}

	tm.app.WriteUnpublishedFile(t, spec.storageID, storedBackup.fileName)
	tm.app.ExecuteSQL(t, "DELETE FROM storages_pending_file_deletions WHERE storage_id = ? AND file_name = ?",
		spec.storageID, storedBackup.fileName)
	tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{DatabaseID: spec.databaseID, StorageID: spec.storageID})
	tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, ?, ?)`, storedBackup.id, spec.databaseID, spec.storageID, spec.status, storedBackup.fileName)

	return storedBackup
}

const refusedBackupDeletionMessage = "backup deletion refused by the test"

// refuseLogicalBackupDeletion makes every deletion of a logical backup record fail, a failure the API
// cannot cause.
func refuseLogicalBackupDeletion(t *testing.T, app *apptest.App) {
	t.Helper()

	app.ExecuteSQL(t, `CREATE FUNCTION refuse_logical_backup_deletion() RETURNS trigger AS $$
		BEGIN RAISE EXCEPTION '`+refusedBackupDeletionMessage+`'; END
		$$ LANGUAGE plpgsql`)
	app.ExecuteSQL(t, `CREATE TRIGGER refuse_logical_backup_deletion BEFORE DELETE ON backups_logical
		FOR EACH ROW EXECUTE FUNCTION refuse_logical_backup_deletion()`)
}
