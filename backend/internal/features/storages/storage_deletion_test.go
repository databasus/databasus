package storages

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storage_files "databasus-backend/internal/features/storages/files"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/users"
	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func Test_DeleteStorage_WhenBackupsStillReferenceIt_IsRefused(t *testing.T) {
	fixture := createStorageDeletionFixture(t, nil)
	pendingFileName := writeUnpublishedFile(t, fixture.storage.ID)
	databaseID := insertDatabaseRow(t, fixture.workspaceID)
	backupID := uuid.New()
	executeSQL(t, `INSERT INTO logical_backups (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, 'COMPLETED', 'backup.dump')`, backupID, databaseID, fixture.storage.ID)

	err := GetStorageService().DeleteStorage(t.Context(), fixture.owner, fixture.storage.ID)

	assert.ErrorIs(t, err, ErrStorageInUse,
		"backup rows are the only record of the file names, so the storage cannot go while they exist")
	assert.True(t, isFileInProvider(t, fixture.storage, pendingFileName),
		"a refused deletion removes no file, not even one waiting for cleanup")
	assert.Equal(t, []uuid.UUID{backupID},
		listIDs(t, "SELECT id FROM logical_backups WHERE storage_id = ?", fixture.storage.ID))
	assert.Equal(t, []uuid.UUID{fixture.storage.ID},
		listIDs(t, "SELECT id FROM storages WHERE id = ?", fixture.storage.ID))
}

func Test_DeleteStorage_WhenBackupConfigUsesIt_IsRefused(t *testing.T) {
	fixture := createStorageDeletionFixture(t, nil)
	pendingFileName := writeUnpublishedFile(t, fixture.storage.ID)
	databaseID := insertDatabaseRow(t, fixture.workspaceID)
	executeSQL(t, `INSERT INTO logical_backup_configs (database_id, storage_id, send_notifications_on, interval_type)
		VALUES (?, ?, '', 'DAILY')`, databaseID, fixture.storage.ID)

	err := GetStorageService().DeleteStorage(t.Context(), fixture.owner, fixture.storage.ID)

	assert.ErrorIs(t, err, ErrStorageInUse)
	assert.Equal(t, []uuid.UUID{fixture.storage.ID},
		listIDs(t, "SELECT storage_id FROM logical_backup_configs WHERE database_id = ?", databaseID))
	assert.True(t, isFileInProvider(t, fixture.storage, pendingFileName),
		"a refused deletion removes no file, not even one waiting for cleanup")
}

func Test_DeleteStorage_WhenCleanupIsPending_DrainsItBeforeTheCredentialsGo(t *testing.T) {
	fixture := createStorageDeletionFixture(t, nil)
	pendingFileName := writeUnpublishedFile(t, fixture.storage.ID)

	require.NoError(t, GetStorageService().DeleteStorage(t.Context(), fixture.owner, fixture.storage.ID))

	assert.False(t, isFileInProvider(t, fixture.storage, pendingFileName),
		"the drain runs while the credentials still exist, so the file goes with the storage")
	assert.Empty(t, listIDs(t, "SELECT id FROM storages WHERE id = ?", fixture.storage.ID))
}

func Test_DeleteStorage_WhenWriteIsStillRunning_LeavesTheFileAndNamesIt(t *testing.T) {
	uploadStarted := make(chan struct{})
	releaseUpload := make(chan struct{})
	var signalUploadStarted, signalRelease sync.Once
	var providerRequestCount atomic.Int64

	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		providerRequestCount.Add(1)
		signalUploadStarted.Do(func() { close(uploadStarted) })
		<-releaseUpload
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	release := func() { signalRelease.Do(func() { close(releaseUpload) }) }
	t.Cleanup(provider.Close)
	t.Cleanup(release)

	fixture := createStorageDeletionFixture(t, func(workspaceID uuid.UUID) *Storage {
		return CreateTestFlakyS3Storage(workspaceID, provider.URL)
	})
	runningFileName := "running-" + uuid.New().String()

	writeFinished := make(chan error, 1)
	go func() {
		_, err := GetStorageFileStore().WriteFile(
			context.WithoutCancel(t.Context()),
			storage_files.StoredFileReference{StorageID: fixture.storage.ID, FileName: runningFileName},
			strings.NewReader("still uploading"),
		)
		writeFinished <- err
	}()

	testwait.Until(t, "the upload reaches the provider", func(context.Context) error {
		select {
		case <-uploadStarted:
			return nil
		default:
			return errors.New("the upload has not started")
		}
	})

	require.NoError(t, GetStorageService().DeleteStorage(t.Context(), fixture.owner, fixture.storage.ID))

	assert.Equal(t, int64(1), providerRequestCount.Load(),
		"the deletion leaves the running write's file alone instead of asking the provider to remove it")

	release()
	assert.Error(t, <-writeFinished)

	assert.Contains(t, listAuditMessages(t, fixture.workspaceID),
		fmt.Sprintf("Storage %s deleted with 1 file(s) left behind: %s", fixture.storage.Name, runningFileName))
}

func Test_DeleteStorage_WhenProviderIsUnreachable_SucceedsAndNamesTheFilesLeftBehind(t *testing.T) {
	fixture := createStorageDeletionFixture(t, func(workspaceID uuid.UUID) *Storage {
		return CreateTestFlakyS3Storage(workspaceID, "http://127.0.0.1:1")
	})

	leftBehindCount := maxLeftBehindFileNamesInAudit + 2
	references := make([]storage_files.StoredFileReference, 0, leftBehindCount)
	fileNames := make([]string, 0, leftBehindCount)

	for index := range leftBehindCount {
		fileName := fmt.Sprintf("unreachable-%02d", index)
		fileNames = append(fileNames, fileName)
		references = append(references, storage_files.StoredFileReference{
			StorageID: fixture.storage.ID,
			FileName:  fileName,
		})
	}

	require.NoError(t, db.GetTransactor().Transaction(t.Context(), func(ctx context.Context) error {
		return GetStorageFileStore().RequestFileDeletions(ctx, references)
	}))

	require.NoError(t, GetStorageService().DeleteStorage(t.Context(), fixture.owner, fixture.storage.ID))

	assert.Empty(t, listIDs(t, "SELECT id FROM storages WHERE id = ?", fixture.storage.ID))
	assert.Contains(t, listAuditMessages(t, fixture.workspaceID), fmt.Sprintf(
		"Storage %s deleted with %d file(s) left behind: %s and 2 more",
		fixture.storage.Name, leftBehindCount, strings.Join(fileNames[:maxLeftBehindFileNamesInAudit], ", "),
	))
}

type storageDeletionFixture struct {
	owner       users.Actor
	workspaceID uuid.UUID
	storage     *Storage
}

// createStorage defaults to a local storage when nil. Cleanup deletes a storage the test left in
// place the way the application does, so the files it still waits to clean up go with it.
func createStorageDeletionFixture(
	t *testing.T,
	createStorage func(workspaceID uuid.UUID) *Storage,
) storageDeletionFixture {
	t.Helper()

	if createStorage == nil {
		createStorage = CreateTestStorage
	}

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Storage Deletion Workspace", owner)
	storage := createStorage(workspace.ID)

	t.Cleanup(func() {
		executeSQL(t, "DELETE FROM databases WHERE workspace_id = ?", workspace.ID)

		if len(listIDs(t, "SELECT id FROM storages WHERE id = ?", storage.ID)) > 0 {
			require.NoError(t, GetStorageService().DeleteStorage(
				context.WithoutCancel(t.Context()),
				users.Actor{UserID: owner.UserID, Role: users.RoleMember},
				storage.ID,
			))
		}

		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	})

	return storageDeletionFixture{
		owner:       users.Actor{UserID: owner.UserID, Role: users.RoleMember},
		workspaceID: workspace.ID,
		storage:     storage,
	}
}

// writeUnpublishedFile leaves the file with a pending deletion, as a backup that never committed
// does.
func writeUnpublishedFile(t *testing.T, storageID uuid.UUID) string {
	t.Helper()

	fileName := "pending-" + uuid.New().String()

	_, err := GetStorageFileStore().WriteFile(
		t.Context(),
		storage_files.StoredFileReference{StorageID: storageID, FileName: fileName},
		strings.NewReader("waiting for cleanup"),
	)
	require.NoError(t, err)

	return fileName
}

func insertDatabaseRow(t *testing.T, workspaceID uuid.UUID) uuid.UUID {
	t.Helper()

	databaseID := uuid.New()
	executeSQL(t, "INSERT INTO databases (id, name, type, workspace_id) VALUES (?, 'payments', 'POSTGRES', ?)",
		databaseID, workspaceID)

	return databaseID
}

func isFileInProvider(t *testing.T, storage *Storage, fileName string) bool {
	t.Helper()

	file, err := storage.GetFile(t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), fileName)
	if err != nil {
		return false
	}

	_ = file.Close()

	return true
}

func listAuditMessages(t *testing.T, workspaceID uuid.UUID) []string {
	t.Helper()

	var messages []string
	require.NoError(t, db.GetDb().WithContext(t.Context()).
		Raw("SELECT message FROM lib_audit_events WHERE workspace_id = ? ORDER BY created_at", workspaceID).
		Scan(&messages).Error)

	return messages
}

func listIDs(t *testing.T, query string, values ...any) []uuid.UUID {
	t.Helper()

	var ids []uuid.UUID
	require.NoError(t, db.GetDb().WithContext(context.WithoutCancel(t.Context())).
		Raw(query, values...).Scan(&ids).Error)

	return ids
}

func executeSQL(t *testing.T, statement string, values ...any) {
	t.Helper()

	require.NoError(t, db.GetDb().WithContext(context.WithoutCancel(t.Context())).Exec(statement, values...).Error)
}
