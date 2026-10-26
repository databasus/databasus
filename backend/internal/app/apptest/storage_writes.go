package apptest

import (
	"context"
	"io"

	"databasus-backend/internal/modules/storages"
)

// WriteStorageFile writes content to the storage through the file store and publishes nothing. It
// returns when content ends, so a test that feeds it through a pipe holds the write open, and it
// returns its error instead of failing the test, so a test may call it from a goroutine.
func (a *App) WriteStorageFile(ctx context.Context, reference storages.StoredFileReference, content io.Reader) error {
	_, err := a.app.WriteStorageFile(ctx, reference, content)

	return err
}
