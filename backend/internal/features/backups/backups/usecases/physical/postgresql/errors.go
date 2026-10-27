package usecases_physical_postgresql

import "errors"

var (
	errSourceWalPositionUnavailable = errors.New("source WAL position is unavailable during recovery")

	errStreamedDatabaseDeleted = errors.New("the streamed database is deleted")
	errSegmentClaimLost        = errors.New("the claim of the wal segment is gone")
)
