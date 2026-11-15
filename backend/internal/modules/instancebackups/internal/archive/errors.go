package archive

import "errors"

var (
	// ErrNotDatabasusBackup is a file without the header of this format.
	ErrNotDatabasusBackup = errors.New("the file is not a Databasus backup")
	// ErrMasterKeyMismatch is a first chunk that fails authentication, which is what a wrong key
	// looks like.
	ErrMasterKeyMismatch = errors.New("the master key does not decrypt the Databasus backup")
	// ErrArchiveIncomplete is a file that ends before its END record.
	ErrArchiveIncomplete = errors.New("the Databasus backup file is incomplete")
	// ErrArchiveCorrupted is a malformed record, a dump that differs from END or data after END.
	ErrArchiveCorrupted = errors.New("the Databasus backup file is corrupted")
	errWriterFinished   = errors.New("the Databasus backup writer is already finished")
)
