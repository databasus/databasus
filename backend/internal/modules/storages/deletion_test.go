package storages_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages"
)

func TestDeleteStorage(t *testing.T) {
	t.Run("returns 204 and no longer lists the storage when nothing references it", func(t *testing.T) {
		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 409 in_use and keeps the storage when a logical backup config uses it", func(t *testing.T) {
		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: tm.app.InsertDatabase(t, tm.workspace.Id),
			StorageID:  createdStorage.Id,
		})

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 409 in_use and keeps the storage when a physical backup config uses it", func(t *testing.T) {
		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.InsertPhysicalBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: databaseID,
			StorageID:  createdStorage.Id,
		})

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 409 in_use and removes no file when backups are still stored in it", func(t *testing.T) {
		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		pendingFileName := "pending-" + uuid.NewString()
		tm.app.WriteUnpublishedFile(t, createdStorage.Id, pendingFileName)
		backupID := insertCompletedBackup(t, tm, tm.app.InsertDatabase(t, tm.workspace.Id), createdStorage.Id)

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))
		assert.Equal(t, 1, countPendingDeletions(t, tm.app, createdStorage.Id))
		assert.Equal(t, 1, countRows(t, tm.app, "SELECT COUNT(*) FROM backups_logical WHERE id = ?", backupID))
	})

	t.Run("removes the files waiting for cleanup before the deletion answers", func(t *testing.T) {
		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		pendingFileName := "pending-" + uuid.NewString()
		tm.app.WriteUnpublishedFile(t, createdStorage.Id, pendingFileName)
		require.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))
		assert.Equal(t, 0, countPendingDeletions(t, tm.app, createdStorage.Id))
	})

	t.Run("deletes the storage and names the files left behind when its provider is unreachable",
		func(t *testing.T) {
			tm := newTeam(t)
			storageID := insertStoredS3Storage(t, tm, storedS3Storage{
				bucket:             minioBucket{name: "unreachable", endpointURL: "http://127.0.0.1:1", client: nil},
				encryptedAccessKey: tm.app.EncryptSecret(t, "access"),
				encryptedSecretKey: tm.app.EncryptSecret(t, "secret"),
			})
			leftBehindFileName := "unreachable-" + uuid.NewString()
			tm.app.ExecuteSQL(t, "INSERT INTO storages_pending_deletions (storage_id, file_name) VALUES (?, ?)",
				storageID, leftBehindFileName)

			response := deleteStorage(t, tm.owner, storageID)

			require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
			assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
			assert.Contains(t, listAuditMessages(t, tm, tm.workspace.Id),
				"Storage Offsite deleted with 1 file(s) left behind: "+leftBehindFileName)
		})

	t.Run("deletes the storage without removing a file whose write is still running and names it",
		func(t *testing.T) {
			tm := newTeam(t)
			createdStorage := tm.createLocalStorage(t, "Local")
			runningFileName := "running-" + uuid.NewString()
			contentReader, contentWriter := io.Pipe()
			writeResult := make(chan error, 1)

			go func() {
				writeResult <- tm.app.WriteStorageFile(context.WithoutCancel(t.Context()),
					storages.StoredFileReference{StorageID: createdStorage.Id, FileName: runningFileName},
					contentReader)
			}()

			testwait.Until(t, "the running write registers its file", func(context.Context) error {
				if countPendingDeletions(t, tm.app, createdStorage.Id) == 0 {
					return fmt.Errorf("no pending deletion of %s yet", runningFileName)
				}

				return nil
			})

			response := deleteStorage(t, tm.owner, createdStorage.Id)

			require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
			assert.Contains(t, listAuditMessages(t, tm, tm.workspace.Id),
				"Storage Local deleted with 1 file(s) left behind: "+runningFileName)
			assert.True(t, tm.app.IsLocalStorageStagingFilePresent(t, runningFileName),
				"the deletion must leave the file of the running write in place")

			require.NoError(t, contentWriter.Close())
			assert.Error(t, <-writeResult, "a write whose storage was deleted must not hand out a receipt")
		})
}

func TestDeleteStorageAccess(t *testing.T) {
	t.Run("lets a workspace member delete a storage", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		createdStorage := tm.createLocalStorage(t, "Local")

		response := deleteStorage(t, member, createdStorage.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 403 permission_denied and keeps the storage when a viewer deletes it", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createdStorage := tm.createLocalStorage(t, "Local")

		response := deleteStorage(t, viewer, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 404 not_found and keeps the storage when a stranger deletes another team's storage",
		func(t *testing.T) {
			tm := newTeam(t)
			stranger := tm.stranger(t)
			createdStorage := tm.createLocalStorage(t, "Local")

			response := deleteStorage(t, stranger, createdStorage.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
		})

	t.Run("returns 401 unauthenticated and keeps the storage without a session", func(t *testing.T) {
		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")

		response, err := tm.app.NewClient(t).DeleteStorageWithResponse(t.Context(), createdStorage.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})
}

func deleteStorage(t *testing.T, session apptest.Session, storageID uuid.UUID) *apiclient.DeleteStorageResponse {
	t.Helper()

	response, err := session.Client.DeleteStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)

	return response
}

// insertCompletedBackup writes the backup record directly, because a completed backup through the
// contract needs a reachable database.
func insertCompletedBackup(t *testing.T, tm team, databaseID, storageID uuid.UUID) uuid.UUID {
	t.Helper()

	backupID := uuid.New()
	tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, 'COMPLETED', ?)`, backupID, databaseID, storageID, "backup-"+backupID.String())

	return backupID
}

func countPendingDeletions(t *testing.T, app *apptest.App, storageID uuid.UUID) int {
	t.Helper()

	return countRows(t, app, "SELECT COUNT(*) FROM storages_pending_deletions WHERE storage_id = ?", storageID)
}

func countRows(t *testing.T, app *apptest.App, query string, values ...any) int {
	t.Helper()

	var rowCounts []int
	app.QuerySQL(t, &rowCounts, query, values...)
	require.Len(t, rowCounts, 1)

	return rowCounts[0]
}

func listAuditMessages(t *testing.T, tm team, workspaceID uuid.UUID) []string {
	t.Helper()

	auditEvents := apptest.ListAllAuditEvents(t, tm.admin, apiclient.ListAuditEventsParams{
		WorkspaceId:  &workspaceID,
		AuthorUserId: nil,
		Limit:        nil,
		After:        nil,
	})

	messages := make([]string, 0, len(auditEvents))
	for _, auditEvent := range auditEvents {
		messages = append(messages, strings.TrimSpace(auditEvent.Message))
	}

	return messages
}
