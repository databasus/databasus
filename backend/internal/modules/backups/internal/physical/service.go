package physical

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/auditing"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/backuping"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/restorestream"
	"databasus-backend/internal/modules/backups/internal/physical/retention"
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
	"databasus-backend/internal/modules/backups/internal/physical/walstreaming"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Service struct {
	backuping            *backuping.Service
	cleaner              *retention.Cleaner
	databases            databases.Module
	storages             storages.Module
	workspaces           workspaces.Module
	configs              *config.Service
	chains               *chains.Service
	backupRepository     *catalog.BackupRepository
	walSegmentRepository *catalog.WalSegmentRepository
	canceller            *backuping.Canceller
	restoreStreamWriter  *restorestream.Writer
	restoreTokenService  *restorestream.Service
	masterKey            *crypto.MasterKey
	auditing             auditing.Recorder
	transactor           *database.Transactor
	clock                clock.Source
	runs                 *background.Group
	streams              *walstreaming.Manager
	health               *streamhealth.Service
	cancellations        *background.CancellationRegistry
	walStreamers         *catalog.WalStreamerRepository
	logger               *slog.Logger
	appCtx               context.Context
}

func (s *Service) authorizeManage(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (databases.Database, error) {
	return s.authorizeDatabase(ctx, actor, databaseID, workspaces.ActionEditContent)
}

func (s *Service) authorizeDatabase(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	action workspaces.Action,
) (databases.Database, error) {
	listedDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}

	if len(listedDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}

	backupDatabase := listedDatabases[0]
	if err := s.workspaces.Authorize(ctx, actor, backupDatabase.WorkspaceID, action); err != nil {
		var problem *apierror.Problem
		if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
			return databases.Database{}, ErrDatabaseNotFound
		}
		return databases.Database{}, err
	}

	if backupDatabase.Type != databases.TypePostgresqlPhysical {
		return databases.Database{}, ErrNotPhysicalDatabase
	}
	return backupDatabase, nil
}

func (s *Service) resolveBackupDatabaseAndStatus(ctx context.Context,
	backupID uuid.UUID,
) (uuid.UUID, catalog.BackupStatus, bool, error) {
	backup, isFound, err := s.backupRepository.FindByID(ctx, backupID)
	if err != nil || !isFound {
		return uuid.Nil, "", false, err
	}
	return backup.DatabaseID, backup.Status, true, nil
}

func toBackupAccessProblem(err error) error {
	if errors.Is(err, ErrDatabaseNotFound) {
		return ErrBackupNotFound
	}
	return err
}
