package usecases_physical_postgresql

import (
	"strings"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// codecFallbackOrder is the server-side compression preference, highest ratio
// first. The real backup IS the probe: each attempt runs pg_basebackup, and a
// source build that lacks the codec's library rejects it pre-stream, so the
// wasted attempt costs only a sub-second round-trip. `none` needs no library
// and never raises that rejection, so the loop always terminates.
var codecFallbackOrder = []physical_enums.PhysicalBackupCompression{
	physical_enums.PhysicalBackupCompressionZstd,
	physical_enums.PhysicalBackupCompressionGzip,
	physical_enums.PhysicalBackupCompressionNone,
}

// compressFlag maps a codec to pg_basebackup's --compress value. Compression is
// server-side so only ~1/3 of the bytes cross the PG->Databasus link (docs/architecture.md, "Physical backups");
// gzip:6 is the balanced analogue of zstd:5, and none is the no-library floor.
func compressFlag(codec physical_enums.PhysicalBackupCompression) string {
	switch codec {
	case physical_enums.PhysicalBackupCompressionGzip:
		return "server-gzip:6"

	case physical_enums.PhysicalBackupCompressionNone:
		return "none"

	default:
		return "server-zstd:5"
	}
}

// The source server rejects a codec its build lacks inside
// parse_compress_specification (src/common/compression.c), which fills in
// "this build does not support compression with %s" and surfaces as
// "invalid compression specification: ..." — the algorithm-specific sinks
// (basebackup_zstd.c) are never reached. Only the inner clause is matched:
// the outer "invalid compression specification" also covers a rejected
// compression level, which is a configuration bug, not a missing library.
func isCompressionUnsupportedError(stderr []byte) bool {
	return strings.Contains(string(stderr), "does not support compression with")
}
