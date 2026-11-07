package logical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/backups/internal/auditing"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/pipeline"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/backups/internal/usage"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

const (
	defaultBackupPageLimit = 20
	maxBackupPageLimit     = 100
)

type Deps struct {
	Usage         *usage.Service
	Repositories  Repositories
	Transactor    *database.Transactor
	Logger        *slog.Logger
	Clock         clock.Source
	Configs       *config.Service
	Pipeline      *pipeline.Pipeline
	Runs          *background.Group
	Cancellations *background.CancellationRegistry
	Audit         auditing.Recorder
	Workspaces    workspaces.Module
	Databases     databases.Module
	Storages      storages.Module
	Notifiers     notifiers.Module
	Restores      runningRestores
}

// Service starts, runs, lists, cancels and deletes the logical backups of databases.
type Service struct {
	usage         *usage.Service
	repository    *backupRepository
	transactor    *database.Transactor
	logger        *slog.Logger
	clock         clock.Source
	configs       *config.Service
	pipeline      *pipeline.Pipeline
	runs          *background.Group
	cancellations *background.CancellationRegistry
	audit         auditing.Recorder
	workspaces    workspaces.Module
	databases     databases.Module
	storages      storages.Module
	notifiers     notifiers.Module
	restores      runningRestores
	// appCtx is the application's context that Start keeps; every run derives its own from it.
	appCtx                  atomic.Pointer[context.Context]
	completedBackupHandlers []BackupCompletedHandler
	deletingBackupHandlers  []BackupsDeletingHandler
}

// runningRestores tells whether a restore reads a backup. Deleting a backup, retention and the
// deletion of a database leave a backup alone while a restore reads it.
type runningRestores interface {
	IsRestoreRunningForBackups(ctx context.Context, backupIDs []uuid.UUID) (bool, error)
	IsRestoreRunningForDatabase(ctx context.Context, databaseID uuid.UUID) (bool, error)
}

// BackupCompleted is published after the publication of a backup commits.
type BackupCompleted struct {
	BackupID   uuid.UUID
	DatabaseID uuid.UUID
}

// BackupCompletedHandler runs after the publication's commit; its error is logged.
type BackupCompletedHandler func(ctx context.Context, event BackupCompleted) error

func NewService(deps Deps) *Service {
	return &Service{
		usage:                   deps.Usage,
		repository:              deps.Repositories.backups,
		transactor:              deps.Transactor,
		logger:                  deps.Logger,
		clock:                   deps.Clock,
		configs:                 deps.Configs,
		pipeline:                deps.Pipeline,
		runs:                    deps.Runs,
		cancellations:           deps.Cancellations,
		audit:                   deps.Audit,
		workspaces:              deps.Workspaces,
		databases:               deps.Databases,
		storages:                deps.Storages,
		notifiers:               deps.Notifiers,
		restores:                deps.Restores,
		appCtx:                  atomic.Pointer[context.Context]{},
		completedBackupHandlers: nil,
		deletingBackupHandlers:  nil,
	}
}

// SubscribeToBackupCompleted adds a handler after the ones subscribed before it. internal/app
// subscribes while it builds the application.
func (s *Service) SubscribeToBackupCompleted(handler BackupCompletedHandler) {
	s.completedBackupHandlers = append(s.completedBackupHandlers, handler)
}

func (s *Service) SubscribeToBackupsDeleting(handler BackupsDeletingHandler) {
	s.deletingBackupHandlers = append(s.deletingBackupHandlers, handler)
}

// StartBackup starts a manual backup. The view right is enough, as it always was: a viewer may take
// a backup but not delete one.
func (s *Service) StartBackup(ctx context.Context, actor users.Actor, databaseID uuid.UUID) (LogicalBackup, error) {
	backupDatabase, err := s.getDatabase(ctx, databaseID)
	if err != nil {
		return LogicalBackup{}, err
	}

	err = s.authorizeDatabase(ctx, actor, backupDatabase, workspaces.ActionView)
	if err != nil {
		return LogicalBackup{}, err
	}

	if !s.pipeline.IsLogical(backupDatabase.Type) {
		return LogicalBackup{}, ErrPhysicalDatabase
	}

	return s.startBackup(ctx, backupDatabase, func(ctx context.Context) error {
		return s.audit.Record(ctx, actor, backupDatabase.WorkspaceID,
			fmt.Sprintf("Backup manually initiated for database: %s", backupDatabase.Name))
	})
}

func (s *Service) CancelBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) (LogicalBackup, error) {
	backup, backupDatabase, err := s.GetAuthorizedBackup(ctx, actor, backupID, workspaces.ActionEditContent)
	if err != nil {
		return LogicalBackup{}, err
	}

	if backup.Status != StatusInProgress {
		return LogicalBackup{}, ErrBackupNotRunning
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		return s.audit.Record(ctx, actor, backupDatabase.WorkspaceID,
			fmt.Sprintf("Backup cancelled for database: %s", backupDatabase.Name))
	})
	if err != nil {
		return LogicalBackup{}, err
	}

	// A run that ended between the read and here is gone from the registry, and its row already
	// holds its outcome.
	s.cancellations.Cancel(backupID)

	return backup, nil
}

// DeleteBackup removes the backup's row and requests the deletion of its files in one transaction.
// The row is locked first, so a restore that starts meanwhile waits on its foreign key check and
// then finds the backup gone.
func (s *Service) DeleteBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) error {
	_, backupDatabase, err := s.GetAuthorizedBackup(ctx, actor, backupID, workspaces.ActionEditContent)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		lockedBackup, isFound, err := s.repository.getForUpdate(ctx, backupID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrBackupNotFound
		}

		if lockedBackup.Status == StatusInProgress {
			return ErrRunningBackupDeletion
		}

		isReadByRestore, err := s.restores.IsRestoreRunningForBackups(ctx, []uuid.UUID{backupID})
		if err != nil {
			return err
		}

		if isReadByRestore {
			return ErrBackupReadByRestore
		}

		err = s.deleteBackupsWithFiles(ctx, []uuid.UUID{backupID})
		if err != nil {
			return err
		}

		return s.audit.Record(ctx, actor, backupDatabase.WorkspaceID,
			fmt.Sprintf("Backup deleted for database: %s", backupDatabase.Name))
	})
}

func (s *Service) GetBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) (LogicalBackup, error) {
	backup, _, err := s.GetAuthorizedBackup(ctx, actor, backupID, workspaces.ActionView)

	return backup, err
}

// ListQuery filters the backups of one database. Each filter is nil when the request leaves it out.
type ListQuery struct {
	DatabaseID    uuid.UUID
	Statuses      []Status
	CreatedBefore *time.Time
	Limit         *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

// Page holds backups newest first.
type Page struct {
	TotalUsageMb float64
	Backups      []LogicalBackup
	// NextCursor is nil on the last page.
	NextCursor *string
}

func (s *Service) ListBackups(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	limit := defaultBackupPageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxBackupPageLimit {
		return Page{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxBackupPageLimit),
		}})
	}

	filter := backupListFilter{
		databaseID:    query.DatabaseID,
		statuses:      query.Statuses,
		createdBefore: query.CreatedBefore,
		after:         nil,
		limit:         limit + 1,
	}

	if query.After != nil {
		afterCursor, err := database.DecodeTimeCursor(*query.After)
		if err != nil {
			return Page{}, err
		}

		filter.after = &afterCursor
	}

	backupDatabase, err := s.getDatabase(ctx, query.DatabaseID)
	if err != nil {
		return Page{}, err
	}

	err = s.authorizeDatabase(ctx, actor, backupDatabase, workspaces.ActionView)
	if err != nil {
		return Page{}, err
	}

	listedBackups, err := s.repository.list(ctx, filter)
	if err != nil {
		return Page{}, err
	}

	totalUsageMb, err := s.usage.GetDatabaseUsage(ctx, query.DatabaseID)
	if err != nil {
		return Page{}, err
	}

	page := Page{Backups: listedBackups[:min(len(listedBackups), limit)], NextCursor: nil, TotalUsageMb: totalUsageMb}

	if len(listedBackups) > limit {
		lastBackup := page.Backups[len(page.Backups)-1]
		nextCursor := database.TimeCursor{Time: lastBackup.CreatedAt, ID: lastBackup.ID}.Encode()
		page.NextCursor = &nextCursor
	}

	return page, nil
}

// GetAuthorizedBackup reads the backup with its database and checks that the actor may take the
// action on it. A backup of a workspace the actor is not a member of answers like a missing one.
func (s *Service) GetAuthorizedBackup(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
	action workspaces.Action,
) (LogicalBackup, databases.Database, error) {
	backup, isFound, err := s.repository.get(ctx, backupID)
	if err != nil {
		return LogicalBackup{}, databases.Database{}, err
	}

	if !isFound {
		return LogicalBackup{}, databases.Database{}, ErrBackupNotFound
	}

	backupDatabase, err := s.getDatabase(ctx, backup.DatabaseID)
	if errors.Is(err, ErrDatabaseNotFound) {
		return LogicalBackup{}, databases.Database{}, ErrBackupNotFound
	}

	if err != nil {
		return LogicalBackup{}, databases.Database{}, err
	}

	err = s.workspaces.Authorize(ctx, actor, backupDatabase.WorkspaceID, action)
	if isNotFound(err) {
		return LogicalBackup{}, databases.Database{}, ErrBackupNotFound
	}

	if err != nil {
		return LogicalBackup{}, databases.Database{}, err
	}

	return backup, backupDatabase, nil
}

// startBackup inserts the backup under the config's share lock, so a save that moves the config to
// another storage either commits first and the backup writes to the new storage, or sees this row
// and refuses. recordAudit runs in the same transaction; the scheduler has no audit event.
func (s *Service) startBackup(
	ctx context.Context,
	backupDatabase databases.Database,
	recordAudit func(ctx context.Context) error,
) (LogicalBackup, error) {
	var startedBackup LogicalBackup

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		backupConfig, err := s.configs.LockAndGetLogicalBackupConfig(ctx, backupDatabase.ID)
		if err != nil {
			return err
		}

		if backupConfig.StorageID == nil {
			return ErrNoStorage
		}

		now := s.clock.Now().UTC()
		backupID := uuid.New()

		startedBackup = LogicalBackup{
			ID:                        backupID,
			DatabaseID:                backupDatabase.ID,
			StorageID:                 *backupConfig.StorageID,
			FileName:                  newFileName(backupDatabase.Name, backupID, now),
			Status:                    StatusInProgress,
			FailMessage:               nil,
			BackupSizeMb:              0,
			BackupRawDbSizeMb:         0,
			BackupDurationMs:          0,
			Encryption:                config.EncryptionNone,
			EncryptionSalt:            nil,
			EncryptionIV:              nil,
			TimescaledbVersion:        "",
			RestoreVerificationStatus: VerificationStatusNotVerified,
			CreatedAt:                 now,
		}

		err = s.repository.insert(ctx, startedBackup)
		if database.IsUniqueViolation(err, inProgressIndexName) {
			return ErrBackupAlreadyRunning
		}

		if err != nil {
			return err
		}

		return recordAudit(ctx)
	})
	if err != nil {
		return LogicalBackup{}, err
	}

	s.launchRun(ctx, startedBackup, backupDatabase)

	return startedBackup, nil
}

// launchRun hands the committed backup to the group under a run context derived from the
// application's, so the request that started it can end without ending the backup.
func (s *Service) launchRun(ctx context.Context, backup LogicalBackup, backupDatabase databases.Database) {
	runCtx, cancel, release := runcontext.New(
		s.getAppContext(ctx),
		runcontext.Limits{
			Shutdown:  runcontext.ErrBackupShutdown,
			Timeout:   runcontext.ErrBackupTimeout,
			TimeLimit: runcontext.LogicalTimeLimit,
		},
	)
	runCtx = logging.With(runCtx, "backup_id", backup.ID, "database_id", backup.DatabaseID)

	s.cancellations.Register(backup.ID, cancel)

	s.runs.Go(runCtx, "logical backup", func(runCtx context.Context) error {
		defer release()
		defer s.cancellations.Unregister(backup.ID)

		s.runBackup(runCtx, backup, backupDatabase)

		return nil
	})
}

// getAppContext is the context Start kept. Before Start, which only a test can reach, it is the
// caller's context without its cancellation.
func (s *Service) getAppContext(ctx context.Context) context.Context {
	appCtx := s.appCtx.Load()
	if appCtx == nil {
		return context.WithoutCancel(ctx)
	}

	return *appCtx
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
		return ErrDatabaseNotFound
	}

	return err
}

func (s *Service) getDatabase(ctx context.Context, databaseID uuid.UUID) (databases.Database, error) {
	foundDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}

	if len(foundDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}

	return foundDatabases[0], nil
}

// deleteBackupsWithFiles removes the rows and requests the deletion of their files in the caller's
// transaction, so the files go only if the rows do.
func (s *Service) deleteBackupsWithFiles(ctx context.Context, backupIDs []uuid.UUID) error {
	removedBackups, err := s.deleteBackupRecords(ctx, backupIDs)
	if err != nil {
		return err
	}
	return s.requestFileDeletions(ctx, removedBackups)
}

func (s *Service) deleteBackupRecords(ctx context.Context, backupIDs []uuid.UUID) ([]LogicalBackup, error) {
	if len(backupIDs) == 0 {
		return nil, nil
	}

	for _, handler := range s.deletingBackupHandlers {
		if err := handler(ctx, BackupsDeleting{BackupIDs: backupIDs}); err != nil {
			return nil, fmt.Errorf("handle logical backups deleting: %w", err)
		}
	}

	deletedBackups, err := s.repository.deleteByIDs(ctx, backupIDs)
	if err != nil {
		return nil, err
	}

	return deletedBackups, nil
}

func (s *Service) requestFileDeletions(ctx context.Context, removedBackups []LogicalBackup) error {
	if len(removedBackups) == 0 {
		return nil
	}

	storedFiles := make([]storages.StoredFileReference, 0, 2*len(removedBackups))
	for _, removedBackup := range removedBackups {
		storedFiles = append(storedFiles, removedBackup.getStoredFiles()...)
	}

	return s.storages.RequestFileDeletions(ctx, storedFiles)
}

func isNotFound(err error) bool {
	var problem *apierror.Problem

	return errors.As(err, &problem) && problem.Code == apierror.CodeNotFound
}
