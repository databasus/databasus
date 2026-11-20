package physical

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/users"
)

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
	database, err := s.authorizeManage(ctx, actor, request.DatabaseID)
	if err != nil {
		return err
	}

	backupConfig, err := s.getConfigAbleToBackUp(ctx, request.DatabaseID)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		requestedBackup, err := s.requestBackupOfKind(ctx, request.Kind, backupConfig)
		if err != nil {
			return err
		}

		return s.auditing.Record(ctx, actor, database.WorkspaceID,
			fmt.Sprintf("%s requested for database: %s", requestedBackup, database.Name))
	})
}

func (s *Service) getConfigAbleToBackUp(
	ctx context.Context,
	databaseID uuid.UUID,
) (config.PhysicalBackupConfig, error) {
	backupConfig, err := s.configs.GetPhysicalBackupConfig(ctx, databaseID)
	if err != nil {
		return config.PhysicalBackupConfig{}, err
	}

	if !backupConfig.IsBackupsEnabled {
		return config.PhysicalBackupConfig{}, errBackupsTurnedOff
	}

	if backupConfig.StorageID == nil {
		return config.PhysicalBackupConfig{}, errNoStorage
	}

	return backupConfig, nil
}

func (s *Service) requestBackupOfKind(
	ctx context.Context,
	kind BackupStartKind,
	backupConfig config.PhysicalBackupConfig,
) (string, error) {
	switch kind {
	case BackupStartAuto:
		return "Physical backup", s.requestSchedulerChoice(ctx, backupConfig)
	case BackupStartFull:
		return "Physical full backup", s.configs.RequestFullBackup(ctx, backupConfig.DatabaseID)
	case BackupStartIncremental:
		return "Physical incremental backup", s.requestIncrementalOnExtendableChain(ctx, backupConfig.DatabaseID)
	default:
		return "", errInvalidBackupKind
	}
}

func (s *Service) requestIncrementalOnExtendableChain(ctx context.Context, databaseID uuid.UUID) error {
	extendable, err := s.chains.FindLastExtendableChainByDatabase(ctx, databaseID)
	if err != nil {
		return err
	}

	if extendable == nil {
		return ErrNoExtendableChain
	}

	return s.configs.RequestIncrementalBackup(ctx, databaseID)
}

func (s *Service) requestSchedulerChoice(ctx context.Context, backupConfig config.PhysicalBackupConfig) error {
	if !backupConfig.BackupType.IsRequiringWalSummary() {
		return s.configs.RequestFullBackup(ctx, backupConfig.DatabaseID)
	}

	extendable, err := s.chains.FindLastExtendableChainByDatabase(ctx, backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	if extendable == nil {
		return s.configs.RequestFullBackup(ctx, backupConfig.DatabaseID)
	}

	return s.configs.RequestIncrementalBackup(ctx, backupConfig.DatabaseID)
}
