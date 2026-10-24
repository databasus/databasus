package app

import (
	"errors"
	"fmt"
	"os"
)

var ErrUnknownConsoleCommand = errors.New("unknown console command")

// StorageProbeError tells the operator which paths the process could not write to and how to fix
// their permissions.
type StorageProbeError struct {
	TempFolder string
	DataFolder string
	Operation  string
	Err        error
}

func (e *StorageProbeError) Error() string {
	return fmt.Sprintf(
		"Databasus cannot write to local storage paths %s and %s as UID %d and GID %d.\n"+
			"Required operation: %s.\n"+
			"Set PUID and PGID or fix the mounted directory permissions: %s\n"+
			"Details: %v",
		e.TempFolder,
		e.DataFolder,
		os.Geteuid(),
		os.Getegid(),
		e.Operation,
		storagePermissionsDocumentationURL,
		e.Err,
	)
}

func (e *StorageProbeError) Unwrap() error {
	return e.Err
}
