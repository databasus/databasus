package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/backoff"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/storages"
)

const (
	DeliveryJobName     = "instancebackups_copy_delivery"
	DeliveryJobInterval = 30 * time.Second
	firstRetryDelay     = time.Minute
	maxRetryDelay       = 15 * time.Minute
	fileNamePrefix      = "databasus-backups-"
)

// RunDeliveryPass uploads every waiting copy, each to its own storage at once, so a slow or broken
// storage never delays another. It first ends the copies whose spool file expired.
func (s *Service) RunDeliveryPass(ctx context.Context, now time.Time) error {
	if err := s.expireSpools(ctx, now); err != nil {
		return err
	}

	dueCopies, err := s.copies.listDue(ctx, now)
	if err != nil || len(dueCopies) == 0 {
		return err
	}

	config, err := s.configs.GetConfig(ctx)
	if err != nil {
		return err
	}

	uploads := background.New(background.Deps{Logger: s.logger})

	for _, dueCopy := range dueCopies {
		uploads.Go(ctx, "instance backup copy delivery", func(uploadCtx context.Context) error {
			return s.deliver(uploadCtx, dueCopy, config.ArchiveNamespace)
		})
	}

	return uploads.Wait(context.WithoutCancel(ctx))
}

func (s *Service) deliver(ctx context.Context, dueCopy Copy, namespace uuid.UUID) error {
	ctx = logging.With(ctx,
		slog.String("instance_backup_id", dueCopy.BackupID.String()),
		slog.String("instance_backup_copy_id", dueCopy.ID.String()))

	backup, isFound, err := s.backups.get(ctx, dueCopy.BackupID)
	if err != nil || !isFound {
		return err
	}

	destination, isCurrent, err := s.getCurrentDestination(ctx, dueCopy)
	if err != nil {
		return err
	}

	if !isCurrent {
		return s.endCopy(ctx, dueCopy, CopyStatusFailed, "the destination was removed before the copy was stored")
	}

	if backup.SpoolPath == nil {
		return s.endCopy(ctx, dueCopy, CopyStatusFailed, "the local file of the backup is no longer kept")
	}

	fileName := getFileName(namespace, backup.ID)

	isStarted, err := s.copies.startAttempt(ctx, dueCopy.ID, fileName)
	if err != nil || !isStarted {
		return err
	}

	receipt, err := s.upload(ctx, *backup.SpoolPath, storages.StoredFileReference{
		StorageID: destination.StorageID,
		FileName:  fileName,
	})
	if errors.Is(err, errSpoolFileMissing) {
		return s.endLostSpool(ctx, dueCopy)
	}

	if err == nil {
		err = s.publish(ctx, dueCopy, backup, receipt)
	}

	if err != nil {
		if ctx.Err() != nil {
			return nil
		}

		return s.retry(ctx, dueCopy, destination, err)
	}

	return s.finishBackupIfEnded(ctx, backup.ID)
}

func (s *Service) upload(
	ctx context.Context,
	spoolPath string,
	reference storages.StoredFileReference,
) (storages.WriteReceipt, error) {
	file, err := os.Open(spoolPath)
	if errors.Is(err, fs.ErrNotExist) {
		return storages.WriteReceipt{}, errSpoolFileMissing
	}

	if err != nil {
		return storages.WriteReceipt{}, fmt.Errorf("open the Databasus backup spool file: %w", err)
	}

	defer func() { _ = file.Close() }()

	return s.storages.WriteFile(ctx, reference, file)
}

// publish confirms the uploaded file only while its destination is still the selection the copy was
// planned for; otherwise the copy fails and the file store removes the unconfirmed file.
func (s *Service) publish(ctx context.Context, dueCopy Copy, backup Backup, receipt storages.WriteReceipt) error {
	isRejected := false

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		isCurrent, err := s.configs.RecordDelivery(txCtx, configs.DeliveryOutcome{
			StorageID:           *dueCopy.StorageID,
			SelectionGeneration: dueCopy.SelectionGeneration,
			SnapshotAt:          *backup.SnapshotAt,
			LastError:           nil,
		})
		if err != nil {
			return err
		}

		if !isCurrent {
			isRejected = true

			return s.copies.end(txCtx, dueCopy.ID, CopyStatusFailed,
				"the destination was removed while the copy was uploaded")
		}

		if err := s.storages.ConfirmFileWrites(txCtx, []storages.WriteReceipt{receipt}); err != nil {
			return err
		}

		return s.copies.recordStored(txCtx, dueCopy.ID, s.clock.Now())
	})
	if err != nil {
		return err
	}

	if isRejected {
		s.logger.InfoContext(ctx, "databasus backup copy rejected after its destination was removed")
	} else {
		s.logger.InfoContext(ctx, "databasus backup copy stored")
	}

	return nil
}

func (s *Service) retry(ctx context.Context, dueCopy Copy, destination configs.Destination, deliveryErr error) error {
	s.logger.WarnContext(ctx, "databasus backup copy failed", slog.Any("error", deliveryErr))

	lastError := logging.RedactError(deliveryErr, maxPersistedErrorRunes)
	nextAttemptAt := s.clock.Now().
		Add(backoff.Exponential{First: firstRetryDelay, Max: maxRetryDelay}.GetDelay(dueCopy.AttemptCount + 1))

	return s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.copies.scheduleRetry(txCtx, dueCopy.ID, nextAttemptAt, lastError); err != nil {
			return err
		}

		_, err := s.configs.RecordDelivery(txCtx, configs.DeliveryOutcome{
			StorageID:           destination.StorageID,
			SelectionGeneration: destination.SelectionGeneration,
			SnapshotAt:          time.Time{},
			LastError:           &lastError,
		})

		return err
	})
}

// endLostSpool ends every copy of a backup whose spool file disappeared and asks for a new dump.
func (s *Service) endLostSpool(ctx context.Context, dueCopy Copy) error {
	s.logger.WarnContext(ctx, "databasus backup spool file is missing")

	err := s.copies.endUnfinished(ctx, dueCopy.BackupID, CopyStatusInterrupted,
		"the local file of the backup was lost before the copy was stored")
	if err != nil {
		return err
	}

	if err := s.configs.RequestRun(ctx); err != nil {
		return err
	}

	s.WakeProduction()

	return s.finishBackupIfEnded(ctx, dueCopy.BackupID)
}

func (s *Service) endCopy(ctx context.Context, dueCopy Copy, status CopyStatus, reason string) error {
	if err := s.copies.end(ctx, dueCopy.ID, status, reason); err != nil {
		return err
	}

	return s.finishBackupIfEnded(ctx, dueCopy.BackupID)
}

// expireSpools fails the copies still waiting for a spool file older than its lifetime; the next
// dump delivers fresher data instead.
func (s *Service) expireSpools(ctx context.Context, now time.Time) error {
	spooledBackups, err := s.backups.listSpooled(ctx)
	if err != nil {
		return err
	}

	for _, backup := range spooledBackups {
		if backup.SpoolExpiresAt == nil || now.Before(*backup.SpoolExpiresAt) {
			continue
		}

		err := s.copies.endUnfinished(ctx, backup.ID, CopyStatusFailed,
			"the copy was not stored within 24 hours of the dump")
		if err != nil {
			return err
		}

		if err := s.finishBackupIfEnded(ctx, backup.ID); err != nil {
			return err
		}
	}

	return nil
}

// finishBackupIfEnded records the outcome once every copy ended and removes the spool file.
func (s *Service) finishBackupIfEnded(ctx context.Context, backupID uuid.UUID) error {
	backup, isFound, err := s.backups.get(ctx, backupID)
	if err != nil || !isFound || backup.Status != StatusInProgress || backup.DumpStatus != DumpStatusSucceeded {
		return err
	}

	copies, err := s.copies.listByBackupIDs(ctx, []uuid.UUID{backupID})
	if err != nil {
		return err
	}

	for _, backupCopy := range copies {
		if !backupCopy.isEnded() {
			return nil
		}
	}

	if err := s.backups.finish(ctx, backupID, getOutcome(copies), s.clock.Now()); err != nil {
		return err
	}

	if backup.SpoolPath != nil {
		removeSpoolFile(ctx, s.logger, *backup.SpoolPath)
	}

	return nil
}

func (s *Service) getCurrentDestination(ctx context.Context, dueCopy Copy) (configs.Destination, bool, error) {
	if dueCopy.StorageID == nil {
		return configs.Destination{}, false, nil
	}

	destinations, err := s.configs.GetDestinations(ctx)
	if err != nil {
		return configs.Destination{}, false, err
	}

	for _, destination := range destinations {
		if destination.StorageID == *dueCopy.StorageID &&
			destination.SelectionGeneration == dueCopy.SelectionGeneration {
			return destination, true, nil
		}
	}

	return configs.Destination{}, false, nil
}

// getFileName is new for every attempt, so a late upload of an earlier attempt never overwrites the
// file of a later one. The namespace keeps installations sharing a storage apart.
func getFileName(namespace, backupID uuid.UUID) string {
	return fmt.Sprintf("%s%s-%s-%s%s", fileNamePrefix, namespace, backupID, uuid.New(), archive.FileExtension)
}

func removeSpoolFile(ctx context.Context, logger *slog.Logger, path string) {
	err := os.Remove(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.WarnContext(ctx, "databasus backup spool file could not be removed", slog.Any("error", err))
	}
}
