package files

import "errors"

var (
	ErrWriteProtectionLost = errors.New("storage file write protection was lost")
	// ErrFileNameAlreadyRegistered means that either a cascading removal reached the file before its
	// write started, or a second writer picked the same name, which per-attempt naming makes a defect.
	ErrFileNameAlreadyRegistered = errors.New("file name is already registered for cleanup")

	// ErrFileTakenForDeletion means the upload finished, but something claimed the file while it ran,
	// so the caller must not publish it.
	ErrFileTakenForDeletion = errors.New("file was taken for deletion during the write")

	ErrReceiptNoLongerValid = errors.New("write receipt no longer matches its pending deletion")

	ErrTransactionRequired = errors.New("publication and deletion requests need the caller's transaction")

	// ErrBackgroundWriteStopped is what a background write reports when its upload ended without a
	// result, such as after a panic.
	ErrBackgroundWriteStopped = errors.New("the background write stopped before the upload finished")
)
