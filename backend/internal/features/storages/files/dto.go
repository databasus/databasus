package storage_files

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/util/encryption"
)

// Dependencies is what both the store and the worker need, so the two
// constructors cannot drift apart or be wired in a different order.
type Dependencies struct {
	Repository     *PendingDeletionRepository
	Locator        StorageLocator
	FieldEncryptor encryption.FieldEncryptor
	Logger         *slog.Logger
	Timings        Timings
}

type ClaimRequest struct {
	Limit        int
	AttemptLease time.Duration
	// ExcludedIDs are the writes running in this process, which have an
	// obligation but no owner willing to let it go yet.
	ExcludedIDs []uuid.UUID
}

type DrainRequest struct {
	Remover          FileRemover
	PendingDeletions []PendingDeletion
	Budget           time.Duration
}

type PendingDeletionSummary struct {
	Pending     int64
	Overdue     int64
	OldestAgeMs int64
}

// Backup code owns the file name; this package owns what happens to the file.
type StoredFileReference struct {
	StorageID uuid.UUID
	FileName  string
}

// Permission to publish one uploaded file. Generation is the pending row's
// generation when the upload finished: anything that takes the file away from its
// writer raises it, so a receipt that no longer matches cannot confirm.
type WriteReceipt struct {
	PendingDeletionID uuid.UUID
	Reference         StoredFileReference
	Generation        int
}

// FileStore is what a caller needs to write one file and get a receipt for it.
// Store satisfies it, and so does any narrower seam a feature declares.
type FileStore interface {
	WriteFile(ctx context.Context, reference StoredFileReference, file io.Reader) (WriteReceipt, error)
}

type FileWriter interface {
	SaveFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
		file io.Reader,
	) error
}

// The returned reader may keep fetching from the provider while it is read, so ctx must outlive the
// stream, not just the call.
type FileReader interface {
	GetFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
	) (io.ReadCloser, error)
}

// StorageReader opens files of one storage. The returned reader may keep fetching from the provider
// while it is read, so ctx must outlive the stream.
type StorageReader interface {
	OpenFile(ctx context.Context, fileName string) (io.ReadCloser, error)
}

type FileRemover interface {
	DeleteFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
	) error
}

// The parent storages package implements this, which keeps provider selection
// there and this package free of an import cycle.
type StorageLocator interface {
	GetFileWriter(ctx context.Context, storageID uuid.UUID) (FileWriter, error)
	GetFileReader(ctx context.Context, storageID uuid.UUID) (FileReader, error)
	GetFileRemover(ctx context.Context, storageID uuid.UUID) (FileRemover, error)
}
