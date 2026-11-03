package catalog

import (
	"context"
	"errors"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type BackupLifecycleDeps struct {
	Configs    *config.Service
	Backups    *BackupRepository
	Databases  databases.Module
	Storages   storages.Module
	Transactor *database.Transactor
	Clock      clock.Source
}

type BackupLifecycle struct {
	configs    *config.Service
	backups    *BackupRepository
	databases  databases.Module
	storages   storages.Module
	transactor *database.Transactor
	clock      clock.Source
}

func NewBackupLifecycle(deps BackupLifecycleDeps) *BackupLifecycle {
	return &BackupLifecycle{
		configs: deps.Configs, backups: deps.Backups, databases: deps.Databases, storages: deps.Storages,
		transactor: deps.Transactor, clock: deps.Clock,
	}
}

func (s *BackupLifecycle) Start(ctx context.Context, backup *Backup) (bool, error) {
	var isInserted bool
	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.configs.CheckStoragePublication(
			ctx,
			config.StoragePublication{
				DatabaseID: backup.DatabaseID,
				StorageID:  backup.StorageID,
				Generation: backup.StorageGeneration,
			},
		); err != nil {
			return err
		}

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
				if err := s.configs.CheckStoragePublication(
					ctx,
					config.StoragePublication{
						DatabaseID: spec.Backup.DatabaseID,
						StorageID:  spec.Backup.StorageID,
						Generation: spec.Backup.StorageGeneration,
					},
				); err != nil {
					return err
				}

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

			if !isBackupUnpublishable(err) {
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

		if err != nil && !isBackupUnpublishable(err) {
			return err
		}

		return s.storages.RequestFileDeletions(ctx, backupFileReferences(spec.Backup))
	})
	return isChanged, err
}

// isBackupUnpublishable answers true when the backup can no longer become completed: another
// transition ended its row, or its storage changed. Its files then only wait for deletion.
func isBackupUnpublishable(err error) bool {
	return errors.Is(err, errBackupSuperseded) || errors.Is(err, config.ErrStaleStoragePublication)
}

func backupFileReferences(backup *Backup) []storages.StoredFileReference {
	return artifactReferences(backup.StorageID, getFileName(backup.FileName), backup.GetManifestFileName())
}
