package pipeline

import (
	"fmt"

	io_utils "databasus-backend/internal/lib/iox"
)

type backupStreamTrackers struct {
	storageRead   *io_utils.FailureTrackingReader
	decodedStream *io_utils.FailureTrackingReader
	// isDownloadedBeforeRestore means every read of the stream happens before the restore tool starts.
	isDownloadedBeforeRestore bool
}

// getBackupStreamFailure must be consulted before the restore tool's own exit error: a truncated
// download makes the tool fail on the half-written statement it was fed, and that downstream syntax
// error would otherwise be reported as the cause.
//
// The transport failure wins over the decode failure for the same reason: a severed download
// surfaces one layer up as a decompression error, and reporting that would hide the real cause.
func getBackupStreamFailure(trackers backupStreamTrackers) error {
	readError := trackers.storageRead.GetFirstReadError()
	if readError == nil {
		readError = trackers.decodedStream.GetFirstReadError()
	}

	if readError == nil {
		return nil
	}

	if trackers.isDownloadedBeforeRestore {
		return fmt.Errorf("%w: %w", errBackupDownloadFailed, readError)
	}

	return fmt.Errorf("%w: %w", errBackupStreamFailed, readError)
}
