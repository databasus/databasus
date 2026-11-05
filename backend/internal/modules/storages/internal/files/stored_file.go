package files

import "github.com/google/uuid"

// StoredFileReference names a file in a storage. Backup code owns the file name; this package owns
// what happens to the file.
type StoredFileReference struct {
	StorageID uuid.UUID
	FileName  string
}

// WriteReceipt is the permission to publish one uploaded file. Generation is the pending row's
// generation when the upload finished: anything that takes the file away from its writer raises it,
// so a receipt that no longer matches cannot confirm.
type WriteReceipt struct {
	PendingDeletionID uuid.UUID
	Reference         StoredFileReference
	Generation        int
}
