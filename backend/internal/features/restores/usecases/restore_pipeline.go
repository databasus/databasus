package usecases

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/klauspost/compress/zstd"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/lib/crypto"
	io_utils "databasus-backend/internal/util/io"
)

const restoreTimeLimit = 23 * time.Hour

// restoreFromStorage streams the backup file from the storage, decrypts and decompresses it, and
// hands it to the engine. Two trackers record the first failed read of the storage and of the
// decoded stream, because the restore tool fails on the half statement a broken stream fed it, and
// that error would otherwise be reported as the cause.
func (uc *RestoreBackupUsecase) restoreFromStorage(
	parentCtx context.Context,
	run restoreRun,
	restoreTarget logicalRestoreTarget,
) error {
	ctx, cancel := context.WithTimeout(parentCtx, restoreTimeLimit)
	defer cancel()

	backupFile, err := run.storageReader.OpenFile(ctx, run.backup.FileName)
	if err != nil {
		return fmt.Errorf("failed to get backup file from storage: %w", err)
	}
	defer func() {
		closeErr := backupFile.Close()
		if closeErr != nil {
			run.logger.ErrorContext(parentCtx, "failed to close backup reader", "error", closeErr)
		}
	}()

	storageRead := io_utils.NewFailureTrackingReader(backupFile)

	decryptedStream, err := uc.decryptIfEncrypted(storageRead, run.backup)
	if err != nil {
		return err
	}

	decodedStream := decryptedStream

	if restoreTarget.GetStoredCompression().isZstd {
		zstdDecoder, err := zstd.NewReader(decryptedStream)
		if err != nil {
			return fmt.Errorf("failed to create zstd reader: %w", err)
		}
		defer zstdDecoder.Close()

		decodedStream = zstdDecoder
	}

	decodedStreamRead := io_utils.NewFailureTrackingReader(decodedStream)

	restoreErr := restoreTarget.Restore(ctx, decodedStreamRead)

	if errors.Is(ctx.Err(), context.Canceled) {
		return errors.New("restore cancelled")
	}

	streamErr := restores_core.GetBackupStreamFailure(restores_core.BackupStreamTrackers{
		StorageRead:   storageRead,
		DecodedStream: decodedStreamRead,
	})
	if streamErr != nil {
		return streamErr
	}

	return restoreErr
}

// decryptIfEncrypted reads the header of an encrypted file at once, so a file swapped for another
// backup's fails before the restore tool receives any data.
func (uc *RestoreBackupUsecase) decryptIfEncrypted(
	storedStream io.Reader,
	backup *backups_core_logical.LogicalBackup,
) (io.Reader, error) {
	if backup.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		return storedStream, nil
	}

	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		return nil, errors.New("backup is encrypted but missing encryption metadata")
	}

	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption IV: %w", err)
	}

	decryptingReader, err := crypto.NewDecryptingReader(
		storedStream, uc.masterKey, backup.ID, crypto.StreamHeader{Salt: salt, Nonce: nonce},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	return decryptingReader, nil
}
