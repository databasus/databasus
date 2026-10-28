package backuping

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

// Bounds for the post-success PUT of the reconstructed `.manifest` sidecar.
// serializeManifest emits ~210 bytes per source file, so a 500k-file cluster
// produces a ~100 MB sidecar that a slow S3-compatible endpoint cannot absorb
// under a flat deadline — and losing the sidecar marks an already-stored backup
// CHAIN_BROKEN. The floor matches walSegmentUploadTimeout and covers endpoint
// latency on the common few-MB manifest; the ceiling keeps a wedged endpoint
// from pinning a backup worker indefinitely.
const (
	manifestSaveMinTimeout            = 2 * time.Minute
	manifestSaveMaxTimeout            = 15 * time.Minute
	manifestSaveThroughputBytesPerSec = 256 * 1024
)

func manifestSaveTimeout(payloadSizeBytes int) time.Duration {
	budget := time.Duration(payloadSizeBytes/manifestSaveThroughputBytesPerSec) * time.Second

	return min(max(budget, manifestSaveMinTimeout), manifestSaveMaxTimeout)
}

// saveManifestSidecar uploads the reconstructed manifest. When encryption is on
// it uses a FRESH salt+nonce (NewEncryptingWriter) — never the tar's — because
// the chunk nonce is baseNonce||chunkIndex, so reusing the tar's (salt, nonce)
// would reuse (key, nonce) on chunk 0 (a catastrophic GCM failure). The fresh
// salt/IV are returned (base64) for persistence so the INCR consumer can decrypt.
type manifestSidecarSpec struct {
	Params   runStreamParams
	FileName string
	Manifest []byte
}

func saveManifestSidecar(ctx context.Context, spec manifestSidecarSpec) (manifestSidecar, error) {
	p := spec.Params
	manifestBytes := spec.Manifest

	var (
		saltBase64  string
		nonceBase64 string
	)

	payload := bytes.NewReader(manifestBytes)
	payloadSizeBytes := len(manifestBytes)

	if p.Encryption == catalog.BackupEncryptionEncrypted {
		var encrypted bytes.Buffer

		encryptingWriter, header, setupErr := crypto.NewEncryptingWriter(&encrypted, p.MasterKey, p.BackupID)
		if setupErr != nil {
			return manifestSidecar{}, fmt.Errorf("setup manifest encryption: %w", setupErr)
		}

		if _, writeErr := encryptingWriter.Write(manifestBytes); writeErr != nil {
			return manifestSidecar{}, fmt.Errorf("encrypt manifest: %w", writeErr)
		}

		if closeErr := encryptingWriter.Close(); closeErr != nil {
			return manifestSidecar{}, fmt.Errorf("flush manifest encryption: %w", closeErr)
		}

		payload = bytes.NewReader(encrypted.Bytes())
		payloadSizeBytes = encrypted.Len()
		saltBase64 = base64.StdEncoding.EncodeToString(header.Salt)
		nonceBase64 = base64.StdEncoding.EncodeToString(header.Nonce)
	}

	// The deadline starts here, after encryption, so CPU-bound work never eats the
	// upload's budget, and it is sized from the bytes that actually go over the wire.
	saveTimeout := manifestSaveTimeout(payloadSizeBytes)

	p.Logger.DebugContext(
		ctx,
		"uploading manifest sidecar",
		slog.Any("size_mb", float64(payloadSizeBytes)/(1024*1024)),
		slog.Any("timeout", saveTimeout),
		slog.Any("backup_id", p.BackupID),
	)

	saveCtx, cancel := context.WithTimeout(ctx, saveTimeout)
	defer cancel()

	receipt, err := p.FileStore.WriteFile(
		saveCtx,
		storages.StoredFileReference{StorageID: p.StorageID, FileName: spec.FileName},
		payload,
	)
	if err != nil {
		return manifestSidecar{}, err
	}

	return manifestSidecar{Receipt: receipt, SaltBase64: saltBase64, NonceBase64: nonceBase64}, nil
}

// manifestSidecar is what the manifest upload produced: the receipt that keeps the
// file and the encryption material its catalog row records.
type manifestSidecar struct {
	Receipt     storages.WriteReceipt
	SaltBase64  string
	NonceBase64 string
}
