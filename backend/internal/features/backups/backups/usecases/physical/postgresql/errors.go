package usecases_physical_postgresql

import "errors"

// errByteStall and errManifestWalk are carried as the cancel cause on the
// per-stream context (context.WithCancelCause) so post-stream classification can
// tell WHY pg_basebackup was aborted. Three independent sources can cancel the
// stream — the byte-stall watcher, a storage-save failure, and the manifest
// goroutine — and context.Canceled alone cannot distinguish them; without
// distinct causes a manifest failure would masquerade as a network stall.
var (
	errByteStall    = errors.New("byte-stall watcher cancelled pg_basebackup")
	errManifestWalk = errors.New("manifest reconstruction failed")

	errStreamedDatabaseDeleted = errors.New("the streamed database is deleted")
	errSegmentClaimLost        = errors.New("the claim of the wal segment is gone")
)
