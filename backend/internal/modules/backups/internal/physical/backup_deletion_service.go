package physical

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/users"
)

func (s *Service) DeleteBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) error {
	full, _, err := s.fullBackupRepository.FindByID(ctx, backupID)
	if err != nil {
		return err
	}
	if full != nil {
		database, err := s.authorizeManage(ctx, actor, full.DatabaseID)
		if err != nil {
			return toBackupAccessProblem(err)
		}

		if err := s.deleteFull(ctx, full); err != nil {
			return err
		}

		s.recordAuditAfterAction(ctx, actor, database,
			fmt.Sprintf("Physical full backup deleted for database: %s", database.Name))

		return nil
	}

	incremental, _, err := s.incrementalBackupRepository.FindByID(ctx, backupID)
	if err != nil {
		return err
	}
	if incremental != nil {
		database, err := s.authorizeManage(ctx, actor, incremental.DatabaseID)
		if err != nil {
			return toBackupAccessProblem(err)
		}

		if err := s.deleteIncremental(ctx, incremental); err != nil {
			return err
		}

		s.recordAuditAfterAction(ctx, actor, database,
			fmt.Sprintf("Physical incremental backup deleted for database: %s", database.Name))

		return nil
	}

	return ErrBackupNotFound
}

func (s *Service) deleteFull(ctx context.Context, full *catalog.FullBackup) error {
	s.stopInFlightForDelete(ctx, full.DatabaseID, func(inFlightBackupID uuid.UUID) bool {
		if inFlightBackupID == full.ID {
			return true
		}

		incremental, _, err := s.incrementalBackupRepository.FindByID(ctx, inFlightBackupID)

		return err == nil && incremental != nil && incremental.RootFullBackupID == full.ID
	})

	if full.StartLSN == nil {
		return s.fullBackupRepository.DeleteByID(ctx, full.ID)
	}

	_, err := s.chains.DeleteFullCompletely(ctx, full.ID)

	return err
}

func (s *Service) deleteIncremental(
	ctx context.Context,
	target *catalog.IncrementalBackup,
) error {
	s.stopInFlightForDelete(ctx, target.DatabaseID, func(inFlightBackupID uuid.UUID) bool {
		if inFlightBackupID == target.ID {
			return true
		}

		claimed, _, err := s.incrementalBackupRepository.FindByID(ctx, inFlightBackupID)
		if err != nil || claimed == nil {
			return false
		}

		return claimed.RootFullBackupID == target.RootFullBackupID &&
			claimed.StartLSN != nil && target.StartLSN != nil &&
			*claimed.StartLSN >= *target.StartLSN
	})

	_, err := s.chains.DeleteIncrementalCascade(ctx, target.ID)

	return err
}

func (s *Service) stopInFlightForDelete(
	ctx context.Context,
	databaseID uuid.UUID,
	willDelete func(inFlightBackupID uuid.UUID) bool,
) {
	claim, _, err := s.inFlightBackupRepository.FindByDatabaseID(ctx, databaseID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to look up in-flight backup for delete",
			slog.Any("database_id", databaseID),
			slog.Any("error", err),
		)

		return
	}

	if claim == nil {
		return
	}

	if willDelete(claim.BackupID) {
		s.canceller.CancelInFlightForDatabase(ctx, databaseID)
	}
}
