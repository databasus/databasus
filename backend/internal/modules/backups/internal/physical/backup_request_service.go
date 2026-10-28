package physical

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/users"
)

func (s *Service) RequestBackup(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) error {
	database, err := s.authorizeManage(ctx, actor, databaseID)
	if err != nil {
		return err
	}

	config, err := s.configs.GetPhysicalBackupConfig(ctx, databaseID)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.requestSchedulerChoice(ctx, databaseID, config); err != nil {
			return err
		}
		return s.auditing.Record(ctx, actor, database.WorkspaceID,
			fmt.Sprintf("Physical backup requested for database: %s", database.Name))
	})
}

func (s *Service) RequestFullBackup(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) error {
	database, err := s.authorizeManage(ctx, actor, databaseID)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.configs.RequestFullBackup(ctx, databaseID); err != nil {
			return err
		}
		return s.auditing.Record(ctx, actor, database.WorkspaceID,
			fmt.Sprintf("Physical full backup requested for database: %s", database.Name))
	})
}

func (s *Service) RequestIncrementalBackup(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) error {
	database, err := s.authorizeManage(ctx, actor, databaseID)
	if err != nil {
		return err
	}

	extendable, err := s.chains.FindLastExtendableChainByDatabase(ctx, databaseID)
	if err != nil {
		return err
	}

	if extendable == nil {
		return ErrNoExtendableChain
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.configs.RequestIncrementalBackup(ctx, databaseID); err != nil {
			return err
		}
		return s.auditing.Record(ctx, actor, database.WorkspaceID,
			fmt.Sprintf("Physical incremental backup requested for database: %s", database.Name))
	})
}

func (s *Service) requestSchedulerChoice(
	ctx context.Context,
	databaseID uuid.UUID,
	config config.PhysicalBackupConfig,
) error {
	if !config.BackupType.IsRequiringWalSummary() {
		return s.configs.RequestFullBackup(ctx, databaseID)
	}

	extendable, err := s.chains.FindLastExtendableChainByDatabase(ctx, databaseID)
	if err != nil {
		return err
	}

	if extendable == nil {
		return s.configs.RequestFullBackup(ctx, databaseID)
	}

	return s.configs.RequestIncrementalBackup(ctx, databaseID)
}

type BackupStartKind string

const (
	BackupStartAuto        BackupStartKind = "AUTO"
	BackupStartFull        BackupStartKind = "FULL"
	BackupStartIncremental BackupStartKind = "INCREMENTAL"
)

type BackupStart struct {
	DatabaseID uuid.UUID
	Kind       BackupStartKind
}

func (s *Service) StartBackup(ctx context.Context, actor users.Actor, request BackupStart) error {
	switch request.Kind {
	case BackupStartAuto:
		return s.RequestBackup(ctx, actor, request.DatabaseID)
	case BackupStartFull:
		return s.RequestFullBackup(ctx, actor, request.DatabaseID)
	case BackupStartIncremental:
		return s.RequestIncrementalBackup(ctx, actor, request.DatabaseID)
	default:
		return errInvalidBackupKind
	}
}
