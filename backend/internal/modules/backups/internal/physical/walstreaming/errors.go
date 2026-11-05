package walstreaming

import "errors"

var (
	errStorageNotFound              = errors.New("storage not found")
	errSourceWalPositionUnavailable = errors.New("source WAL position is unavailable during recovery")

	errStreamedDatabaseDeleted = errors.New("the streamed database is deleted")
	errSegmentClaimLost        = errors.New("the claim of the wal segment is gone")
)
