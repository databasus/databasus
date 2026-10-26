package app

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrUnknownConsoleCommand = errors.New("unknown console command")
	ErrFatalClientTools      = errors.New("client tools of a fatal bundle are broken")
)

// StorageProbeError tells the operator which paths the process could not write to and how to fix
// their permissions.
type StorageProbeError struct {
	TempFolder string
	DataFolder string
	// Err names the operation that failed.
	Err error
}

func (e *StorageProbeError) Error() string {
	return fmt.Sprintf(
		"Databasus cannot write to local storage paths %s and %s as UID %d and GID %d.\n"+
			"Failed operation: %v.\n"+
			"Set PUID and PGID or fix the mounted directory permissions: %s",
		e.TempFolder,
		e.DataFolder,
		os.Geteuid(),
		os.Getegid(),
		e.Err,
		storagePermissionsDocumentationURL,
	)
}

func (e *StorageProbeError) Unwrap() error {
	return e.Err
}
