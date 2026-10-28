//go:build testhelpers

package storages_bridge

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/storage"
)

// CreateTestStorage inserts a local storage, as the old tests need a storage without a workspace
// member who could create one through the contract.
func CreateTestStorage(workspaceID uuid.UUID) storages.Storage {
	createdStorage := storages.Storage{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        "Test Storage " + uuid.NewString(),
		Type:        storages.TypeLocal,
	}

	insertTestStorage(createdStorage)
	execute("INSERT INTO storages_local (storage_id) VALUES (?)", createdStorage.ID)

	return createdStorage
}

// RemoveTestStorage deletes the storage row; its provider row and its pending deletions go with it.
// The context is stripped of cancellation because callers pass t.Context() from a t.Cleanup, and the
// test context is already cancelled by the time cleanup runs.
func RemoveTestStorage(ctx context.Context, storageID uuid.UUID) {
	err := storage.GetDb().WithContext(context.WithoutCancel(ctx)).
		Exec("DELETE FROM storages WHERE id = ?", storageID).Error
	if err != nil {
		panic(err)
	}
}

func insertTestStorage(createdStorage storages.Storage) {
	execute("INSERT INTO storages (id, workspace_id, name, type) VALUES (?, ?, ?, ?)",
		createdStorage.ID, createdStorage.WorkspaceID, createdStorage.Name, createdStorage.Type)
}

func execute(statement string, values ...any) {
	err := storage.GetDb().Exec(statement, values...).Error
	if err != nil {
		panic(err)
	}
}
