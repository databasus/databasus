package apptest

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

// getTestFileTimings keep the windows short, so a test moves the manual clock a few seconds to make a
// file due. AttemptLease bounds a real provider call, so it stays long enough for a slow machine.
func getTestFileTimings() storages.FileTimings {
	return storages.FileTimings{
		CommitWindow:   2 * time.Second,
		AttemptLease:   10 * time.Second,
		RetryBaseDelay: 10 * time.Millisecond,
		RetryMaxDelay:  50 * time.Millisecond,
		WorkerTick:     10 * time.Millisecond,
		ClaimBatchSize: 10,
	}
}

// WriteUnpublishedFile writes a file to the storage through the file store and publishes nothing,
// so the file keeps its pending deletion, as the files of a backup that never committed do.
func (a *App) WriteUnpublishedFile(tb testing.TB, storageID uuid.UUID, fileName string) {
	tb.Helper()

	_, err := a.app.WriteStorageFile(
		tb.Context(),
		storages.StoredFileReference{StorageID: storageID, FileName: fileName},
		strings.NewReader("written by "+tb.Name()),
	)
	if err != nil {
		tb.Fatalf("write %s to storage %s: %v", fileName, storageID, err)
	}
}

// IsLocalStorageStagingFilePresent tells whether a local storage still holds the file of a write
// that is running, which it stages in the temp folder until the write ends.
func (a *App) IsLocalStorageStagingFilePresent(tb testing.TB, fileName string) bool {
	tb.Helper()

	return isFilePresent(tb, filepath.Join(a.tempFolder, fileName))
}

// IsLocalStorageFilePresent tells whether a local storage still holds the file, which every local
// storage keeps in the data folder of the application.
func (a *App) IsLocalStorageFilePresent(tb testing.TB, fileName string) bool {
	tb.Helper()

	return isFilePresent(tb, filepath.Join(a.dataFolder, fileName))
}

func isFilePresent(tb testing.TB, filePath string) bool {
	tb.Helper()

	_, err := os.Stat(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}

	if err != nil {
		tb.Fatalf("check the local storage file %s: %v", filePath, err)
	}

	return true
}

func (a *App) GetLocalStorageFilePath(fileName string) string {
	return filepath.Join(a.dataFolder, fileName)
}
func (a *App) GetTempFolder() string { return a.tempFolder }
