package usecases_physical_postgresql

import (
	"context"
	"io"

	"databasus-backend/internal/modules/storages"
)

// FileStore is the part of the storages module a physical backup writes and publishes its files
// through. A test hands in a fake to fail or hold a write that a real provider cannot be made to.
type FileStore interface {
	WriteFile(
		ctx context.Context,
		reference storages.StoredFileReference,
		file io.Reader,
	) (storages.WriteReceipt, error)
	StartBackgroundWrite(ctx context.Context, spec storages.BackgroundWriteSpec) storages.BackgroundWrite
	ConfirmFileWrites(ctx context.Context, receipts []storages.WriteReceipt) error
	RequestFileDeletions(ctx context.Context, references []storages.StoredFileReference) error
}
