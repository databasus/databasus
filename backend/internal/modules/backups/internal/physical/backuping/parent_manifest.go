package backuping

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func downloadParentManifest(
	ctx context.Context,
	spec backupExecutionSpec,
) (manifestPath string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "pgincr_"+uuid.New().String())
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp dir: %w", err)
	}

	cleanupAll := func() { _ = os.RemoveAll(tmpDir) }

	reader, err := spec.StorageReader.OpenFile(ctx, spec.ParentManifest.FileName)
	if err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("fetch parent manifest: %w", err)
	}
	defer func() { _ = reader.Close() }()

	decoded := io.Reader(reader)

	if spec.ParentManifest.Encryption == catalog.BackupEncryptionEncrypted {
		decoded, err = decryptParentManifest(reader, spec)
		if err != nil {
			cleanupAll()

			return "", func() {}, err
		}
	}

	manifestPath = filepath.Join(tmpDir, "backup_manifest")

	out, err := os.Create(manifestPath)
	if err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("create manifest temp file: %w", err)
	}

	if _, err := io.Copy(out, decoded); err != nil {
		_ = out.Close()
		cleanupAll()

		return "", func() {}, fmt.Errorf("copy manifest into temp file: %w", err)
	}

	if err := out.Close(); err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("close manifest temp file: %w", err)
	}

	return manifestPath, cleanupAll, nil
}

func decryptParentManifest(reader io.Reader, spec backupExecutionSpec) (io.Reader, error) {
	salt, err := base64.StdEncoding.DecodeString(spec.ParentManifest.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode parent manifest salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(spec.ParentManifest.IV)
	if err != nil {
		return nil, fmt.Errorf("decode parent manifest nonce: %w", err)
	}

	dec, err := crypto.NewDecryptingReader(
		reader, spec.MasterKey, spec.ParentManifest.BackupID, crypto.StreamHeader{Salt: salt, Nonce: nonce},
	)
	if err != nil {
		return nil, fmt.Errorf("init decryption reader: %w", err)
	}

	return dec, nil
}
