package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestReassignLogicalBackupStorage(t *testing.T) {
	t.Parallel()
	for _, isSameHistoricalStorage := range []bool{false, true} {
		scenario := "removes retained history when another storage is assigned after clearing"
		if isSameHistoricalStorage {
			scenario = "removes retained history when its historical storage is reassigned after clearing"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			oldStorageID := tm.createLocalStorage(t)
			targetStorageID := oldStorageID
			if !isSameHistoricalStorage {
				targetStorageID = tm.createLocalStorage(t)
			}
			saveLogicalBackupConfig(t, tm.owner, databaseID, oldStorageID)
			oldBackup := insertLogicalBackupInStorage(t, tm, databaseID, oldStorageID, "COMPLETED")

			cleared := updateBackupConfig(
				t,
				tm.owner,
				databaseID,
				newLogicalBackupConfigInput(t, nil, newLogicalBackupSettings()),
			)
			require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
			assert.Nil(t, apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId)
			assert.Equal(t, 1, countLogicalBackups(t, tm, databaseID))
			assert.Zero(
				t,
				countRows(
					t,
					tm,
					"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=?",
					databaseID,
				),
			)

			saveLogicalBackupConfig(t, tm.owner, databaseID, targetStorageID)
			assert.Zero(t, countLogicalBackups(t, tm, databaseID))
			assert.Equal(
				t,
				2,
				countRows(
					t,
					tm,
					"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=?",
					databaseID,
				),
			)

			tm.app.RunJobPass(t, storageFileDeletionJobName)
			assert.False(t, tm.app.IsLocalStorageFilePresent(t, oldBackup.fileName))
		})
	}
}

//nolint:tparallel // The subtests share one source server.
func TestRefuseLogicalStorageSwitchDuringWork(t *testing.T) {
	t.Parallel()
	server := startRestoreServer(t, "postgres:17")
	for _, isRestore := range []bool{false, true} {
		for _, isCleared := range []bool{false, true} {
			scenario := "returns in_use with an active backup"
			if isRestore {
				scenario = "returns in_use with an active restore"
			}
			if isCleared {
				scenario += " after clearing storage"
			}
			t.Run(scenario, func(t *testing.T) {
				tm := newTeam(t)
				database := newBackedUpDatabase(t, tm, server)
				if isRestore {
					startBlockedRestore(t, tm.owner, server, database.backup.Id)
				} else {
					startBlockedBackup(t, tm, server, database)
				}
				if isCleared {
					response := updateBackupConfig(
						t,
						tm.owner,
						database.databaseID,
						newLogicalBackupConfigInput(t, nil, newLogicalBackupSettings()),
					)
					require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
				}

				storageID := tm.createLocalStorage(t)
				response := updateBackupConfig(
					t,
					tm.owner,
					database.databaseID,
					newLogicalBackupConfigInput(t, &storageID, newLogicalBackupSettings()),
				)

				requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
				assert.Contains(t, listLogicalBackupIDs(t, tm.owner, database.databaseID), database.backup.Id)
				assert.Zero(
					t,
					countRows(
						t,
						tm,
						"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=?",
						database.databaseID,
					),
				)
			})
		}
	}
}

func TestRollbackStorageSwitch(t *testing.T) {
	t.Parallel()
	for _, shouldFailNotifier := range []bool{false, true} {
		scenario := "keeps the move and catalog unchanged when deletion obligations cannot be saved"
		if shouldFailNotifier {
			scenario = "keeps the move and catalog unchanged when notifier replacement fails"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			sourceStorageID := tm.createLocalStorage(t)
			saveLogicalBackupConfig(t, tm.owner, databaseID, sourceStorageID)
			oldBackup := insertLogicalBackupInStorage(t, tm, databaseID, sourceStorageID, "COMPLETED")
			targetWorkspace := tm.createWorkspace(t)
			targetStorageID := tm.app.CreateLocalStorage(t, tm.owner, targetWorkspace.Id)
			transfer := newDatabaseTransfer(t, targetWorkspace.Id, newStorageSwitchedOnTransfer(t, targetStorageID))

			problemStatus := http.StatusInternalServerError
			problemCode := apiclient.ProblemCodeInternal
			if shouldFailNotifier {
				transfer.TargetNotifierIds = &[]uuid.UUID{uuid.New()}
				problemStatus = http.StatusBadRequest
				problemCode = apiclient.ProblemCodeInvalidRequest
			} else {
				tm.app.ExecuteSQL(
					t,
					`CREATE FUNCTION refuse_switch_deletion() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'deletion refused'; END $$ LANGUAGE plpgsql`,
				)
				tm.app.ExecuteSQL(
					t,
					`CREATE TRIGGER refuse_switch_deletion BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION refuse_switch_deletion()`,
				)
			}

			response := requestDatabaseTransfer(t, tm.owner, databaseID, transfer)
			requireProblem(t, response.StatusCode(), response.Body, problemStatus, problemCode)

			databaseResponse, err := tm.owner.Client.GetDatabaseWithResponse(t.Context(), databaseID)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, databaseResponse.StatusCode())
			assert.Equal(t, tm.workspace.Id, databaseResponse.JSON200.WorkspaceId)
			requireConfigStorage(t, tm, databaseID, sourceStorageID)
			assert.Equal(t, 1, countLogicalBackups(t, tm, databaseID))
			assert.Zero(
				t,
				countRows(
					t,
					tm,
					"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=?",
					databaseID,
				),
			)
			tm.app.RunJobPass(t, storageFileDeletionJobName)
			assert.True(t, tm.app.IsLocalStorageFilePresent(t, oldBackup.fileName))
		})
	}
}

func TestRollbackReassignedLogicalStorage(t *testing.T) {
	t.Parallel()
	t.Run("preserves the cleared config and history when reassignment cleanup cannot be saved", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		oldStorageID, targetStorageID := tm.createLocalStorage(t), tm.createLocalStorage(t)
		saveLogicalBackupConfig(t, tm.owner, databaseID, oldStorageID)
		backup := insertLogicalBackupInStorage(t, tm, databaseID, oldStorageID, "COMPLETED")

		cleared := updateBackupConfig(
			t,
			tm.owner,
			databaseID,
			newLogicalBackupConfigInput(t, nil, newLogicalBackupSettings()),
		)
		require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
		tm.app.RefuseStorageFileDeletions(t)

		assigned := updateBackupConfig(
			t,
			tm.owner,
			databaseID,
			newLogicalBackupConfigInput(t, &targetStorageID, newLogicalBackupSettings()),
		)
		require.Equal(t, http.StatusInternalServerError, assigned.StatusCode(), "%s", assigned.Body)

		assert.Nil(t, apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId)
		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, databaseID), backup.id)
		assert.Zero(
			t,
			countRows(
				t,
				tm,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=?",
				databaseID,
			),
		)
		assert.Zero(
			t,
			countRows(t, tm, "SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=?", oldStorageID),
		)
		tm.app.RunJobPass(t, storageFileDeletionJobName)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, backup.fileName))
	})
}
