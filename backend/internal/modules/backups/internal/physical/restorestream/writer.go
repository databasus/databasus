package restorestream

import (
	"archive/tar"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/storages"
)

// Writer turns a resolved RestoreSet into the single tar stream the user (and,
// later, restore verification) consumes. Every artifact ships as stored — decrypted
// but still compressed: each backup as full/base.tar<ext> and incr-N/base.tar<ext>
// (with its reconstructed backup_manifest beside it), each WAL segment as
// wal/<segment>.zst, history files plaintext as wal/<name>.history. A trailing
// MANIFEST.sha256 lets the recipient verify the transfer. The recovery script that
// decompresses the blobs, folds the chain with pg_combinebackup and arms PITR (it
// takes the target as a CLI argument) is served separately, not shipped in the tar.
// It is transport- and auth-agnostic: callers decide who may stream and where the
// bytes go.
type Writer struct {
	storages storages.Module
}

// Write fails on an encrypted artifact when masterKey is empty, rather than streaming a corrupt
// download.
func (rw *Writer) Write(
	ctx context.Context,
	w io.Writer,
	set *chains.RestoreSet,
	masterKey *crypto.MasterKey,
) error {
	tarWriter := tar.NewWriter(w)

	storageCache := make(map[uuid.UUID]storages.StorageReader)
	checksums := newChecksumLedger()

	fullSize, err := artifactSizeBytes(set.RootFull.ID, set.RootFull.BackupSizeMb)
	if err != nil {
		return fmt.Errorf("stream full backup: %w", err)
	}

	if err := rw.writeBackupDir(ctx, tarWriter, "full", backupArtifact{
		FileName:         deref(set.RootFull.FileName),
		Encryption:       set.RootFull.Encryption,
		EncryptionSalt:   set.RootFull.EncryptionSalt,
		EncryptionIV:     set.RootFull.EncryptionIV,
		ManifestFileName: set.RootFull.ManifestFileName,
		ManifestSalt:     set.RootFull.ManifestEncryptionSalt,
		ManifestIV:       set.RootFull.ManifestEncryptionIV,
		RowID:            set.RootFull.ID,
		StorageID:        set.RootFull.StorageID,
		Compression:      set.RootFull.Compression,
		SizeBytes:        fullSize,
	}, masterKey, storageCache, checksums); err != nil {
		return fmt.Errorf("stream full backup: %w", err)
	}

	for i, incremental := range set.Incrementals {
		incrSize, err := artifactSizeBytes(incremental.ID, incremental.BackupSizeMb)
		if err != nil {
			return fmt.Errorf("stream incremental %s: %w", incremental.ID, err)
		}

		if err := rw.writeBackupDir(ctx, tarWriter, fmt.Sprintf("incr-%d", i+1), backupArtifact{
			FileName:         deref(incremental.FileName),
			Encryption:       incremental.Encryption,
			EncryptionSalt:   incremental.EncryptionSalt,
			EncryptionIV:     incremental.EncryptionIV,
			ManifestFileName: incremental.ManifestFileName,
			ManifestSalt:     incremental.ManifestEncryptionSalt,
			ManifestIV:       incremental.ManifestEncryptionIV,
			RowID:            incremental.ID,
			StorageID:        incremental.StorageID,
			Compression:      incremental.Compression,
			SizeBytes:        incrSize,
		}, masterKey, storageCache, checksums); err != nil {
			return fmt.Errorf("stream incremental %s: %w", incremental.ID, err)
		}
	}

	for _, segment := range set.WalSegments {
		if err := rw.writeWalSegment(ctx, tarWriter, segment, masterKey, storageCache, checksums); err != nil {
			return fmt.Errorf("stream wal segment %s: %w", segment.WalFilename, err)
		}
	}

	for _, history := range set.HistoryFiles {
		if err := rw.writeHistoryFile(ctx, tarWriter, history, masterKey, storageCache, checksums); err != nil {
			return fmt.Errorf("stream history file %s: %w", history.HistoryFilename, err)
		}
	}

	if err := writeTarBytes(tarWriter, "MANIFEST.sha256", 0o644, checksums.render(), checksums.skip()); err != nil {
		return err
	}

	return tarWriter.Close()
}

// The recovery script decompresses the blob into a directory and pg_combinebackup reads
// the manifest from there, one per input directory, so the manifest must ship beside the blob.
func (rw *Writer) writeBackupDir(
	ctx context.Context,
	tarWriter *tar.Writer,
	dirName string,
	artifact backupArtifact,
	masterKey *crypto.MasterKey,
	storageCache map[uuid.UUID]storages.StorageReader,
	checksums *checksumLedger,
) error {
	if artifact.FileName == "" {
		return fmt.Errorf("backup %s has no stored file", artifact.RowID)
	}
	if artifact.ManifestFileName == nil {
		return fmt.Errorf("backup %s has no reconstructed manifest sidecar", artifact.RowID)
	}

	storageReader, err := rw.resolveStorage(ctx, artifact.StorageID, storageCache)
	if err != nil {
		return err
	}

	// Ship as stored (decrypt only, still compressed); the recovery script
	// decompresses on the restore side. Decompressing here would inflate the blob
	// by its compression ratio and blow the download up by that factor.
	reader, cleanup, err := openArtifact(
		ctx,
		storageReader,
		masterKey,
		artifactSource{
			fileName:   artifact.FileName,
			encryption: artifact.Encryption,
			salt:       artifact.EncryptionSalt,
			iv:         artifact.EncryptionIV,
			keyID:      artifact.RowID,
			codec:      catalog.BackupCompressionNone,
		},
	)
	if err != nil {
		return err
	}
	defer cleanup()

	blobName := dirName + "/base.tar" + compressionExtension(artifact.Compression)
	if err := streamSizedTarEntry(tarWriter, blobName, 0o600, artifact.SizeBytes, reader, checksums); err != nil {
		return err
	}

	return rw.writeManifest(ctx, tarWriter, dirName, artifact, masterKey, storageCache, checksums)
}

func (rw *Writer) writeManifest(
	ctx context.Context,
	tarWriter *tar.Writer,
	dirName string,
	artifact backupArtifact,
	masterKey *crypto.MasterKey,
	storageCache map[uuid.UUID]storages.StorageReader,
	checksums *checksumLedger,
) error {
	storageReader, err := rw.resolveStorage(ctx, artifact.StorageID, storageCache)
	if err != nil {
		return err
	}

	// The manifest sidecar is stored as raw bytes (encrypted with the backup's
	// row ID but a fresh salt/IV), never zstd-compressed.
	reader, cleanup, err := openArtifact(
		ctx,
		storageReader,
		masterKey,
		artifactSource{
			fileName:   *artifact.ManifestFileName,
			encryption: artifact.Encryption,
			salt:       artifact.ManifestSalt,
			iv:         artifact.ManifestIV,
			keyID:      artifact.RowID,
			codec:      catalog.BackupCompressionNone,
		},
	)
	if err != nil {
		return err
	}
	defer cleanup()

	return streamTarEntry(tarWriter, dirName+"/backup_manifest", 0o600, reader, checksums)
}

func (rw *Writer) writeWalSegment(
	ctx context.Context,
	tarWriter *tar.Writer,
	segment *catalog.WalSegment,
	masterKey *crypto.MasterKey,
	storageCache map[uuid.UUID]storages.StorageReader,
	checksums *checksumLedger,
) error {
	if segment.FileName == nil {
		return fmt.Errorf("wal segment %s has no stored file", segment.WalFilename)
	}

	storageReader, err := rw.resolveStorage(ctx, segment.StorageID, storageCache)
	if err != nil {
		return err
	}

	// WAL ships as stored (decrypt only, still zstd) under its bare name + .zst;
	// the recovery script decompresses on demand at replay time. Decompressing
	// here would re-inflate every near-empty segment back to a full 16 MB and
	// blow the download up by orders of magnitude.
	reader, cleanup, err := openArtifact(
		ctx,
		storageReader,
		masterKey,
		artifactSource{
			fileName:   *segment.FileName,
			encryption: segment.Encryption,
			salt:       segment.EncryptionSalt,
			iv:         segment.EncryptionIV,
			keyID:      segment.ID,
			codec:      catalog.BackupCompressionNone,
		},
	)
	if err != nil {
		return err
	}
	defer cleanup()

	return streamTarEntry(tarWriter, "wal/"+segment.WalFilename+".zst", 0o600, reader, checksums)
}

func (rw *Writer) writeHistoryFile(
	ctx context.Context,
	tarWriter *tar.Writer,
	history *catalog.WalHistoryFile,
	masterKey *crypto.MasterKey,
	storageCache map[uuid.UUID]storages.StorageReader,
	checksums *checksumLedger,
) error {
	storageReader, err := rw.resolveStorage(ctx, history.StorageID, storageCache)
	if err != nil {
		return err
	}

	// History files keep their encryption parameters only in the .metadata
	// sidecar, not on the catalog row — read it to learn how to decrypt.
	metadata, err := readHistoryMetadata(
		ctx,
		storageReader,
		history.FileName,
	)
	if err != nil {
		return err
	}

	reader, cleanup, err := openArtifact(
		ctx,
		storageReader,
		masterKey,
		artifactSource{
			fileName:   history.FileName,
			encryption: metadata.Encryption,
			salt:       emptyToNil(metadata.EncryptionSalt),
			iv:         emptyToNil(metadata.EncryptionIV),
			keyID:      history.ID,
			codec:      catalog.BackupCompressionZstd,
		},
	)
	if err != nil {
		return err
	}
	defer cleanup()

	return streamTarEntry(tarWriter, "wal/"+history.HistoryFilename, 0o600, reader, checksums)
}

func (rw *Writer) resolveStorage(
	ctx context.Context,
	storageID uuid.UUID,
	storageCache map[uuid.UUID]storages.StorageReader,
) (storages.StorageReader, error) {
	if cached, ok := storageCache[storageID]; ok {
		return cached, nil
	}

	storageReader, err := rw.storages.GetStorageReader(ctx, storageID)
	if err != nil {
		return nil, fmt.Errorf("load storage %s: %w", storageID, err)
	}

	storageCache[storageID] = storageReader

	return storageReader, nil
}
