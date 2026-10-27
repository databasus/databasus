package usecases

import (
	"context"
	"io"
)

// logicalRestoreTarget is the database a backup is restored into, behind the engine of its type.
type logicalRestoreTarget interface {
	GetStoredCompression() storedCompression
	Restore(ctx context.Context, input io.Reader) error
}

// storedCompression is what the pipeline undoes after decrypting a backup file. The tools of
// PostgreSQL and MongoDB read their own compressed archive; a MySQL or MariaDB backup is plain SQL
// that the backup compressed with zstd.
type storedCompression struct {
	isZstd bool
}

var (
	storedCompressionNone = storedCompression{isZstd: false}
	storedCompressionZstd = storedCompression{isZstd: true}
)
