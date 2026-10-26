package apptest

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	old_config "databasus-backend/internal/config"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
)

// WriteUnpublishedFile writes a file to the storage through the file store and publishes nothing,
// so the file keeps its pending deletion, as the files of a backup that never committed do.
func (a *App) WriteUnpublishedFile(tb testing.TB, storageID uuid.UUID, fileName string) {
	tb.Helper()

	_, err := storages.GetStorageFileStore().WriteFile(
		tb.Context(),
		storage_files.StoredFileReference{StorageID: storageID, FileName: fileName},
		strings.NewReader("written by "+tb.Name()),
	)
	if err != nil {
		tb.Fatalf("write %s to storage %s: %v", fileName, storageID, err)
	}
}

// IsLocalStorageFilePresent tells whether a local storage still holds the file, which every local
// storage keeps in the data folder of the application.
func (a *App) IsLocalStorageFilePresent(tb testing.TB, fileName string) bool {
	tb.Helper()

	_, err := os.Stat(filepath.Join(old_config.GetEnv().DataFolder, fileName))
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}

	if err != nil {
		tb.Fatalf("check the local storage file %s: %v", fileName, err)
	}

	return true
}
