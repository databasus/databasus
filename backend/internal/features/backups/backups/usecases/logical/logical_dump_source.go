package usecases_logical

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
	GetStoredCompression() storedCompression
	Dump(ctx context.Context, output io.Writer) error
	Close()
}

// storedCompression is what the pipeline does to a dump before encrypting it. The tools of
// PostgreSQL and MongoDB compress their own output, so it stays as it is; MySQL and MariaDB send
// plain SQL, which the pipeline compresses with zstd.
type storedCompression struct {
	isZstd    bool
	zstdLevel int
}

var storedCompressionNone = storedCompression{isZstd: false, zstdLevel: 0}

func newZstdStoredCompression(level int) storedCompression {
	return storedCompression{isZstd: true, zstdLevel: level}
}
