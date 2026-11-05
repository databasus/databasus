package restorestream

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

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

// backupArtifact is the per-FULL/INCR data the writer needs, flattened so the
// FULL and INCR paths share writeBackupDir.
type backupArtifact struct {
	FileName         string
	Encryption       catalog.BackupEncryption
	EncryptionSalt   *string
	EncryptionIV     *string
	ManifestFileName *string
	ManifestSalt     *string
	ManifestIV       *string
	RowID            uuid.UUID
	StorageID        uuid.UUID
	Compression      catalog.BackupCompression
	SizeBytes        int64
}

// compressionExtension is the suffix the shipped backup blob carries so the
// recovery script knows how to decompress it before pg_combinebackup.
func compressionExtension(codec catalog.BackupCompression) string {
	switch codec {
	case catalog.BackupCompressionGzip:
		return ".gz"
	case catalog.BackupCompressionZstd:
		return ".zst"
	case catalog.BackupCompressionNone:
		return ""
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
// codec is catalog.BackupCompressionNone for objects stored raw (manifests).
type artifactSource struct {
	fileName   string
	encryption catalog.BackupEncryption
	salt, iv   *string
	keyID      uuid.UUID
	codec      catalog.BackupCompression
}

// The returned cleanup tears the decrypt/decompress layers down in order and must be invoked by the
// caller.
func openArtifact(
	ctx context.Context,
	storageReader storages.StorageReader,
	masterKey *crypto.MasterKey,
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

	if src.encryption == catalog.BackupEncryptionEncrypted {
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
	codec catalog.BackupCompression,
) (io.Reader, func(), error) {
	switch codec {
	case catalog.BackupCompressionZstd:
		decoder, err := zstd.NewReader(r)
		if err != nil {
			return nil, nil, err
		}

		return decoder, decoder.Close, nil

	case catalog.BackupCompressionGzip:
		reader, err := gzip.NewReader(r)
		if err != nil {
			return nil, nil, err
		}

		return reader, func() { _ = reader.Close() }, nil

	case catalog.BackupCompressionNone:
		return r, func() {}, nil

	default:
		return nil, nil, fmt.Errorf("unknown compression codec %q", codec)
	}
}

func newDecryptingReader(
	reader io.Reader,
	masterKey *crypto.MasterKey,
	src artifactSource,
) (io.Reader, error) {
	if masterKey == nil {
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

	return crypto.NewDecryptingReader(reader, masterKey, src.keyID, crypto.StreamHeader{Salt: salt, Nonce: iv})
}

func readHistoryMetadata(
	ctx context.Context,
	storageReader storages.StorageReader,
	historyFileName string,
) (*catalog.WalHistoryMetadata, error) {
	reader, err := storageReader.OpenFile(ctx, historyFileName+catalog.MetadataSuffix)
	if err != nil {
		return nil, fmt.Errorf("open history metadata: %w", err)
	}
	defer func() { _ = reader.Close() }()

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read history metadata: %w", err)
	}

	var metadata catalog.WalHistoryMetadata
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
