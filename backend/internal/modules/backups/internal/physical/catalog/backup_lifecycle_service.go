package catalog

import (
	"context"
	"errors"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type BackupLifecycleDeps struct {
	Backups    *BackupRepository
	Databases  databases.Module
	Storages   storages.Module
	Transactor *database.Transactor
	Clock      clock.Source
}

type BackupLifecycle struct {
	backups    *BackupRepository
	databases  databases.Module
	storages   storages.Module
	transactor *database.Transactor
	clock      clock.Source
}

func NewBackupLifecycle(deps BackupLifecycleDeps) *BackupLifecycle {
	return &BackupLifecycle{
		backups: deps.Backups, databases: deps.Databases, storages: deps.Storages,
		transactor: deps.Transactor, clock: deps.Clock,
	}
}

func (s *BackupLifecycle) Start(ctx context.Context, backup *Backup) (bool, error) {
	var isInserted bool
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		backup.Status = BackupStatusInProgress
		var err error
		isInserted, err = s.backups.InsertRunning(ctx, backup)
		return err
	})
	return isInserted, err
}

func (s *BackupLifecycle) Complete(
	ctx context.Context,
	backup *Backup,
	receipts []storages.WriteReceipt,
) (bool, error) {
	return s.transition(ctx, backupTransition{Backup: backup, Receipts: receipts, Status: BackupStatusCompleted})
}

func (s *BackupLifecycle) Fail(ctx context.Context, backup *Backup) (bool, error) {
	return s.transition(ctx, backupTransition{Backup: backup, Status: BackupStatusError})
}

func (s *BackupLifecycle) MarkChainBroken(ctx context.Context, backup *Backup) (bool, error) {
	return s.transition(ctx, backupTransition{Backup: backup, Status: BackupStatusChainBroken})
}

func (s *BackupLifecycle) Cancel(ctx context.Context, backup *Backup) (bool, error) {
	backup.ErrorReason = new(BackupErrorCanceledByUser)
	backup.FailMessage = new(context.Canceled.Error())
	backup.CompletedAt = new(s.clock.Now())
	return s.transition(ctx, backupTransition{Backup: backup, Status: BackupStatusCanceled})
}

func (s *BackupLifecycle) FailLeftInProgress(ctx context.Context) error {
	backups, err := s.backups.FindAllRunning(ctx)
	if err != nil {
		return err
	}
	for _, backup := range backups {
		backup.ErrorReason = new(BackupErrorApplicationRestart)
		backup.FailMessage = new(
			"Backup was interrupted by an application restart and marked failed. Trigger a new backup.",
		)
		backup.CompletedAt = new(s.clock.Now())
		if _, err := s.Fail(ctx, backup); err != nil {
			return err
		}
	}
	return nil
}

type backupTransition struct {
	Backup   *Backup
	Receipts []storages.WriteReceipt
	Status   BackupStatus
}

func (s *BackupLifecycle) transition(ctx context.Context, spec backupTransition) (bool, error) {
	var isChanged bool
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		var err error
		switch spec.Status {
		case BackupStatusCompleted:
			err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
				// Database deletion locks the database before its backups.
				if err := s.databases.RecordBackupSuccess(
					ctx,
					spec.Backup.DatabaseID,
					spec.Backup.CreatedAt,
				); err != nil {
					return err
				}
				var err error
				isChanged, err = s.backups.MarkCompleted(ctx, spec.Backup)
				if err != nil {
					return err
				}
				if !isChanged {
					return errBackupSuperseded
				}
				return s.storages.ConfirmFileWrites(ctx, spec.Receipts)
			})
			if err == nil {
				return nil
			}
			if !errors.Is(err, errBackupSuperseded) {
				return err
			}
		case BackupStatusError:
			isChanged, err = s.backups.MarkFailed(ctx, spec.Backup)
		case BackupStatusChainBroken:
			isChanged, err = s.backups.MarkChainBroken(ctx, spec.Backup)
		case BackupStatusCanceled:
			isChanged, err = s.backups.MarkCanceled(ctx, spec.Backup)
		case BackupStatusInProgress:
			panic("terminal backup transition requires a terminal status")
		}
		if err != nil && !errors.Is(err, errBackupSuperseded) {
			return err
		}

		return s.storages.RequestFileDeletions(ctx, backupFileReferences(spec.Backup))
	})
	return isChanged, err
}

func backupFileReferences(backup *Backup) []storages.StoredFileReference {
	manifestFileName := getFileName(backup.ManifestFileName)
	if manifestFileName == "" && backup.FileName != nil {
		manifestFileName = *backup.FileName + ManifestSuffix
	}
	return artifactReferences(backup.StorageID, getFileName(backup.FileName), manifestFileName)
}
