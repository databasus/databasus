package configsave

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Repositories    config.Repositories
	Configs         *config.Service
	LogicalBackups  *logical.Service
	PhysicalBackups *physical.Service
	Transactor      *database.Transactor
	Logger          *slog.Logger
	Clock           clock.Source
	Workspaces      workspaces.Module
	Storages        storages.Module
	Databases       databases.Module
}

type Service struct {
	repository      *config.Repository
	configs         *config.Service
	logicalBackups  *logical.Service
	physicalBackups *physical.Service
	transactor      *database.Transactor
	logger          *slog.Logger
	clock           clock.Source
	workspaces      workspaces.Module
	storages        storages.Module
	databases       databases.Module
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:      deps.Repositories.Configs,
		configs:         deps.Configs,
		logicalBackups:  deps.LogicalBackups,
		physicalBackups: deps.PhysicalBackups,
		transactor:      deps.Transactor,
		logger:          deps.Logger,
		clock:           deps.Clock,
		workspaces:      deps.Workspaces,
		storages:        deps.Storages,
		databases:       deps.Databases,
	}
}

// A transfer can commit between the initial access check and the locks. The locked database decides
// both storage ownership and cleanup attribution.
func (s *Service) SaveBackupConfig(
	ctx context.Context,
	actor users.Actor,
	savedConfig config.AnyBackupConfig,
) (config.AnyBackupConfig, error) {
	databaseID := savedConfig.GetBackupConfig().DatabaseID

	configDatabase, err := s.getDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	err = s.authorizeDatabase(ctx, actor, configDatabase, workspaces.ActionEditContent)
	if err != nil {
		return nil, err
	}

	var saved config.AnyBackupConfig
	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if _, err := s.configs.LockBackupConfig(ctx, databaseID); err != nil {
			return err
		}

		configDatabase, err := s.getDatabase(ctx, databaseID)
		if err != nil {
			return err
		}
		if err := s.authorizeDatabase(ctx, actor, configDatabase, workspaces.ActionEditContent); err != nil {
			return err
		}

		backupTypes, err := s.databases.GetPhysicalBackupTypes(ctx, []uuid.UUID{databaseID})
		if err != nil {
			return err
		}

		backupType, isPhysical := backupTypes[databaseID]
		scope := storageSwitchScope{WorkspaceID: configDatabase.WorkspaceID}

		switch typedConfig := savedConfig.(type) {
		case config.LogicalBackupConfig:
			if isPhysical {
				return config.ErrMethodMismatch
			}
			saved, err = s.saveLogicalBackupConfig(ctx, scope, typedConfig)
		case config.PhysicalBackupConfig:
			if !isPhysical {
				return config.ErrMethodMismatch
			}
			typedConfig.BackupType = backupType
			saved, err = s.savePhysicalBackupConfig(ctx, scope, typedConfig)
		default:
			return config.ErrMethodMismatch
		}
		if err != nil {
			return err
		}

		configs, err := s.configs.AddCleanupSummaries(ctx, actor, []config.AnyBackupConfig{saved})
		if err != nil {
			return err
		}

		saved = configs[0]

		return nil
	})
	if err != nil {
		return nil, err
	}

	return saved, nil
}

// authorizeDatabase answers a database of a workspace the actor is not a member of like a missing one.
func (s *Service) authorizeDatabase(
	ctx context.Context,
	actor users.Actor,
	authorizedDatabase databases.Database,
	action workspaces.Action,
) error {
	err := s.workspaces.Authorize(ctx, actor, authorizedDatabase.WorkspaceID, action)
	if isNotFound(err) {
		return config.ErrDatabaseNotFound
	}

	return err
}

func isNotFound(err error) bool {
	var problem *apierror.Problem

	return errors.As(err, &problem) && problem.Code == apierror.CodeNotFound
}

func (s *Service) getDatabase(ctx context.Context, databaseID uuid.UUID) (databases.Database, error) {
	foundDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}

	if len(foundDatabases) == 0 {
		return databases.Database{}, config.ErrDatabaseNotFound
	}

	return foundDatabases[0], nil
}
