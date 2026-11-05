package walstreaming

import (
	"context"
	"io"

	"databasus-backend/internal/modules/storages"
)

type FileStore interface {
	WriteFile(
		ctx context.Context,
		reference storages.StoredFileReference,
		file io.Reader,
	) (storages.WriteReceipt, error)
	StartBackgroundWrite(ctx context.Context, spec storages.BackgroundWriteSpec) storages.BackgroundWrite
	ConfirmFileWrites(ctx context.Context, receipts []storages.WriteReceipt) error
	RequestDiscardedFileDeletions(ctx context.Context, request storages.DiscardedFiles) error
	RequestFileDeletions(ctx context.Context, references []storages.StoredFileReference) error
}
