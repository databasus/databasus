// Package archive owns the Databasus backup file. A short plaintext header names the format and
// the backup ID. An encrypted stream follows, sealed with the master key and the backup ID as the
// stream's file ID, whose plaintext is a framed sequence: one MANIFEST record, DUMP records with the
// pg_dump bytes, then one END record with the dump's length and SHA-256. Nothing may follow END.
// The mandatory END record is what makes a file cut between encrypted chunks incomplete, because
// the stream format alone ends cleanly at any chunk boundary.
package archive

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
)

const (
	outerMagic         = "DBSINST1"
	formatVersion      = uint16(1)
	outerHeaderLength  = len(outerMagic) + 2 + 16
	recordHeaderLength = 1 + 4

	recordTypeManifest = byte('M')
	recordTypeDump     = byte('D')
	recordTypeEnd      = byte('E')

	maxManifestLength = 1 << 20
	maxDumpRecordSize = 1 << 20
	endRecordLength   = 16 + 8 + 32
)

// FileExtension is the suffix of a Databasus backup file.
const FileExtension = ".dbbackup"

// Manifest describes the snapshot. It is authenticated by the stream's encryption, so a name or a
// sidecar that claims another release is never trusted over it.
type Manifest struct {
	FormatVersion  int                     `json:"formatVersion"`
	BackupID       uuid.UUID               `json:"backupId"`
	SourceRelease  string                  `json:"sourceRelease"`
	SnapshotAt     time.Time               `json:"snapshotAt"`
	SchemaIdentity database.SchemaIdentity `json:"schemaIdentity"`
}

// Summary is what END records about the dump.
type Summary struct {
	DumpLength uint64
	DumpSha256 [32]byte
}
