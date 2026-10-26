package restore_stream

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	storage_files "databasus-backend/internal/features/storages/files"
)

// historyMetadataSuffix mirrors the ".metadata" sidecar the streamer writes next
// to each history artifact.
const historyMetadataSuffix = ".metadata"

// backupArtifact is the per-FULL/INCR data the writer needs, flattened so the
// FULL and INCR paths share writeBackupDir.
type backupArtifact struct {
	FileName         string
	Encryption       backups_core_enums.BackupEncryption
	EncryptionSalt   *string
	EncryptionIV     *string
	ManifestFileName *string
	ManifestSalt     *string
	ManifestIV       *string
	RowID            uuid.UUID
	StorageID        uuid.UUID
	Compression      physical_enums.PhysicalBackupCompression
	SizeBytes        int64
}

// compressionExtension is the suffix the shipped backup blob carries so the
// recovery script knows how to decompress it before pg_combinebackup.
func compressionExtension(codec physical_enums.PhysicalBackupCompression) string {
	switch codec {
	case physical_enums.PhysicalBackupCompressionGzip:
		return ".gz"
	case physical_enums.PhysicalBackupCompressionZstd:
		return ".zst"
	default:
		return ""
	}
}

// artifactSizeBytes recovers the exact byte length of the stored, decrypt-only
// (still-compressed) artifact from BackupSizeMb so the blob can be streamed into a
// tar entry, whose header needs the size up front, without buffering it. sizeMb is
// bytes/2²⁰ held in a float64; scaling by a power of two is lossless in IEEE-754,
// so the round trip is exact. A completed backup always records it — a nil here is
// a bug, not a zero-byte file.
func artifactSizeBytes(backupID uuid.UUID, sizeMb *float64) (int64, error) {
	if sizeMb == nil {
		return 0, fmt.Errorf("backup %s has no recorded size", backupID)
	}

	return int64(math.Round(*sizeMb * 1024 * 1024)), nil
}

// artifactSource describes how to open one stored object back into plaintext.
// codec is PhysicalBackupCompressionNone for objects stored raw (manifests).
type artifactSource struct {
	fileName   string
	encryption backups_core_enums.BackupEncryption
	salt, iv   *string
	keyID      uuid.UUID
	codec      physical_enums.PhysicalBackupCompression
}

// The returned cleanup tears the decrypt/decompress layers down in order and must be invoked by the
// caller.
func openArtifact(
	ctx context.Context,
	storageReader storage_files.StorageReader,
	masterKey string,
	src artifactSource,
) (io.Reader, func(), error) {
	base, err := storageReader.OpenFile(ctx, src.fileName)
	if err != nil {
		return nil, nil, fmt.Errorf("open %q: %w", src.fileName, err)
	}

	cleanups := []func(){func() { _ = base.Close() }}
	runCleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	reader := io.Reader(base)

	if src.encryption == backups_core_enums.BackupEncryptionEncrypted {
		decryptedReader, err := newDecryptingReader(reader, masterKey, src)
		if err != nil {
			runCleanup()

			return nil, nil, err
		}

		reader = decryptedReader
	}

	decompressed, closeDecompressor, err := newDecompressor(reader, src.codec)
	if err != nil {
		runCleanup()

		return nil, nil, fmt.Errorf("decompress %q: %w", src.fileName, err)
	}

	cleanups = append(cleanups, closeDecompressor)

	return decompressed, runCleanup, nil
}

// newDecompressor wraps r in the reader for codec. The returned closer must be
// called (zstd owns goroutines; gzip holds buffers).
func newDecompressor(
	r io.Reader,
	codec physical_enums.PhysicalBackupCompression,
) (io.Reader, func(), error) {
	switch codec {
	case physical_enums.PhysicalBackupCompressionZstd:
		decoder, err := zstd.NewReader(r)
		if err != nil {
			return nil, nil, err
		}

		return decoder, decoder.Close, nil

	case physical_enums.PhysicalBackupCompressionGzip:
		reader, err := gzip.NewReader(r)
		if err != nil {
			return nil, nil, err
		}

		return reader, func() { _ = reader.Close() }, nil

	case physical_enums.PhysicalBackupCompressionNone:
		return r, func() {}, nil

	default:
		return nil, nil, fmt.Errorf("unknown compression codec %q", codec)
	}
}

func newDecryptingReader(
	reader io.Reader,
	masterKey string,
	src artifactSource,
) (io.Reader, error) {
	if masterKey == "" {
		return nil, fmt.Errorf("artifact %q is encrypted but no master key was provided", src.fileName)
	}
	if src.salt == nil || src.iv == nil {
		return nil, fmt.Errorf("artifact %q is encrypted but missing salt/iv", src.fileName)
	}

	salt, err := base64.StdEncoding.DecodeString(*src.salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt for %q: %w", src.fileName, err)
	}

	iv, err := base64.StdEncoding.DecodeString(*src.iv)
	if err != nil {
		return nil, fmt.Errorf("decode iv for %q: %w", src.fileName, err)
	}

	return backup_encryption.NewDecryptionReader(reader, masterKey, src.keyID, salt, iv)
}

func readHistoryMetadata(
	ctx context.Context,
	storageReader storage_files.StorageReader,
	historyFileName string,
) (*physical_dto.PhysicalWalHistoryMetadata, error) {
	reader, err := storageReader.OpenFile(ctx, historyFileName+historyMetadataSuffix)
	if err != nil {
		return nil, fmt.Errorf("open history metadata: %w", err)
	}
	defer func() { _ = reader.Close() }()

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read history metadata: %w", err)
	}

	var metadata physical_dto.PhysicalWalHistoryMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("parse history metadata: %w", err)
	}

	return &metadata, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func emptyToNil(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
