package pipeline

import (
	"context"
	"io"
)

// logicalDumpSource is one database opened for a dump, behind the engine of its type. Close
// releases what opening it took, such as an SSH tunnel.
type logicalDumpSource interface {
	GetRawSizeMb(ctx context.Context) (float64, bool)
	// GetTimescaleDBVersion is empty for every engine but PostgreSQL, and for a database without
	// the extension.
	GetTimescaleDBVersion(ctx context.Context) string
	Dump(ctx context.Context, output io.Writer) error
	Close()
}

// logicalRestoreSink is a restore target opened for a restore. It is named apart from the restore
// target, which is the settings the user sends.
type logicalRestoreSink interface {
	// IsArchiveFileNeeded reports whether the restore downloads the archive into a file first.
	IsArchiveFileNeeded() bool
	Restore(ctx context.Context, input io.Reader) error
	Close()
}

// storedCompression is what the pipeline does to a dump before encrypting it and undoes after
// decrypting it. The tools of PostgreSQL and MongoDB compress their own output, so it stays as it
// is; MySQL and MariaDB send plain SQL, which the pipeline compresses with zstd.
type storedCompression struct {
	isZstd    bool
	zstdLevel int
}

func getStoredCompressionNone() storedCompression {
	return storedCompression{isZstd: false, zstdLevel: 0}
}

func newZstdStoredCompression(level int) storedCompression {
	return storedCompression{isZstd: true, zstdLevel: level}
}
