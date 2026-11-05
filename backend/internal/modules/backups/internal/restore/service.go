package restore

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
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/shutdown"
	"databasus-backend/internal/modules/backups/internal/auditing"
	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/backups/internal/pipeline"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

const (
	defaultRestorePageLimit = 20
	maxRestorePageLimit     = 100

	restartFailMessage = "Restore failed due to application restart"

	bytesInMegabyte = 1024 * 1024
	// archiveSpaceMargin and minArchiveSpaceBytes size the disk an archive file needs: the backup's
	// size and a tenth more, and never less than one gigabyte.
	archiveSpaceMargin   = 0.1
	minArchiveSpaceBytes = 1024 * bytesInMegabyte
)

type Deps struct {
	ShutdownBudget *shutdown.Budget
	Repositories   Repositories
	Transactor     *database.Transactor
	Logger         *slog.Logger
	Clock          clock.Source
	LogicalBackups *logical.Service
	Pipeline       *pipeline.Pipeline
	Runs           *background.Group
	Cancellations  *background.CancellationRegistry
	Audit          auditing.Recorder
	Workspaces     workspaces.Module
	// TempDir holds the archive of a restore that needs it as a file, so its disk is checked.
	TempDir string
}

// Service starts, follows and cancels the restores of logical backups.
type Service struct {
	shutdownBudget *shutdown.Budget
	repository     *restoreRepository
	transactor     *database.Transactor
	logger         *slog.Logger
	clock          clock.Source
	logicalBackups *logical.Service
	pipeline       *pipeline.Pipeline
	runs           *background.Group
	cancellations  *background.CancellationRegistry
	audit          auditing.Recorder
	workspaces     workspaces.Module
	tempDir        string
	// appCtx is the application's context that Start keeps; every run derives its own from it.
	appCtx atomic.Pointer[context.Context]
}

func NewService(deps Deps) *Service {
	return &Service{
		shutdownBudget: deps.ShutdownBudget,
		repository:     deps.Repositories.restores,
		transactor:     deps.Transactor,
		logger:         deps.Logger,
		clock:          deps.Clock,
		logicalBackups: deps.LogicalBackups,
		pipeline:       deps.Pipeline,
		runs:           deps.Runs,
		cancellations:  deps.Cancellations,
		audit:          deps.Audit,
		workspaces:     deps.Workspaces,
		tempDir:        deps.TempDir,
		appCtx:         atomic.Pointer[context.Context]{},
	}
}

// Start keeps the application's context and fails the restores the previous run left in progress.
// Their targets' credentials were never stored, so none of them can resume.
func (s *Service) Start(appCtx context.Context) error {
	s.appCtx.Store(&appCtx)

	failedCount, err := s.repository.failAllInProgress(appCtx, restartFailMessage)
	if err != nil {
		return err
	}

	if failedCount > 0 {
		s.logger.InfoContext(appCtx, "failed the restores left in progress by the previous run",
			slog.Int64("count", failedCount))
	}

	return nil
}

// StartRestore checks the backup and the target before it answers, so a wrong password or an old
// server version reaches the dialog instead of a failed restore. The opened target, with its
// credentials and its tunnel, lives only in the memory of the run.
func (s *Service) StartRestore(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
	target pipeline.RestoreTarget,
) (Restore, error) {
	backup, backupDatabase, err := s.logicalBackups.GetAuthorizedBackup(ctx, actor, backupID, workspaces.ActionView)
	if err != nil {
		return Restore{}, err
	}

	if backup.Status != logical.StatusCompleted {
		return Restore{}, ErrBackupNotCompleted
	}

	storedBackup, err := backup.GetStoredBackup()
	if err != nil {
		return Restore{}, err
	}

	openedTarget, err := s.openTarget(ctx, backup, backupDatabase, target)
	if err != nil {
		return Restore{}, err
	}

	startedRestore, err := s.insertRestore(ctx, actor, backup, backupDatabase)
	if err != nil {
		openedTarget.Close()

		return Restore{}, err
	}

	s.launchRun(ctx, startedRestore, storedBackup, openedTarget)

	return startedRestore, nil
}

func (s *Service) CancelRestore(ctx context.Context, actor users.Actor, restoreID uuid.UUID) (Restore, error) {
	cancelledRestore, backupDatabase, err := s.getAuthorizedRestore(ctx, actor, restoreID, workspaces.ActionEditContent)
	if err != nil {
		return Restore{}, err
	}

	if cancelledRestore.Status != StatusInProgress {
		return Restore{}, ErrRestoreNotRunning
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		return s.audit.Record(ctx, actor, backupDatabase.WorkspaceID,
			fmt.Sprintf("Restore cancelled for database: %s", backupDatabase.Name))
	})
	if err != nil {
		return Restore{}, err
	}

	// A run that ended between the read and here is gone from the registry, and its row already
	// holds its outcome.
	s.cancellations.Cancel(restoreID)

	return cancelledRestore, nil
}

func (s *Service) GetRestore(ctx context.Context, actor users.Actor, restoreID uuid.UUID) (Restore, error) {
	foundRestore, _, err := s.getAuthorizedRestore(ctx, actor, restoreID, workspaces.ActionView)

	return foundRestore, err
}

type ListQuery struct {
	BackupID uuid.UUID
	Limit    *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

// Page holds restores newest first.
type Page struct {
	Restores []Restore
	// NextCursor is nil on the last page.
	NextCursor *string
}

func (s *Service) ListRestores(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	limit := defaultRestorePageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxRestorePageLimit {
		return Page{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxRestorePageLimit),
		}})
	}

	filter := restoreListFilter{backupID: query.BackupID, after: nil, limit: limit + 1}

	if query.After != nil {
		afterCursor, err := database.DecodeTimeCursor(*query.After)
		if err != nil {
			return Page{}, err
		}

		filter.after = &afterCursor
	}

	_, _, err := s.logicalBackups.GetAuthorizedBackup(ctx, actor, query.BackupID, workspaces.ActionView)
	if err != nil {
		return Page{}, err
	}

	listedRestores, err := s.repository.list(ctx, filter)
	if err != nil {
		return Page{}, err
	}

	page := Page{Restores: listedRestores[:min(len(listedRestores), limit)], NextCursor: nil}

	if len(listedRestores) > limit {
		lastRestore := page.Restores[len(page.Restores)-1]
		nextCursor := database.TimeCursor{Time: lastRestore.CreatedAt, ID: lastRestore.ID}.Encode()
		page.NextCursor = &nextCursor
	}

	return page, nil
}

// openTarget connects to the target and detects its version, then checks the disk of a restore that
// downloads the archive into a file.
func (s *Service) openTarget(
	ctx context.Context,
	backup logical.LogicalBackup,
	backupDatabase databases.Database,
	target pipeline.RestoreTarget,
) (*pipeline.OpenedRestoreTarget, error) {
	openedTarget, err := s.pipeline.OpenRestoreTarget(ctx, pipeline.OpenRestoreTargetSpec{
		SourceDatabaseID:   backup.DatabaseID,
		SourceDatabaseType: backupDatabase.Type,
		Target:             target,
		IsTimescaleDB:      backup.TimescaledbVersion != "",
		Logger:             s.logger,
	})

	var connectionError *pipeline.TargetConnectionError

	var tlsError *pipeline.InvalidTargetTLSError

	switch {
	case err == nil:
	case errors.Is(err, pipeline.ErrTargetOfAnotherEngine):
		return nil, ErrTargetOfAnotherEngine
	case errors.Is(err, pipeline.ErrTargetVersionTooOld):
		return nil, newTargetVersionTooOldError(err)
	case errors.As(err, &connectionError):
		return nil, newTargetConnectionError(connectionError.Err)
	case errors.As(err, &tlsError):
		return nil, newInvalidTargetTLSError(tlsError.Err)
	default:
		return nil, err
	}

	if !openedTarget.IsArchiveFileNeeded() {
		return openedTarget, nil
	}

	err = s.checkArchiveDiskSpace(backup)
	if err != nil {
		openedTarget.Close()

		return nil, err
	}

	return openedTarget, nil
}

func (s *Service) checkArchiveDiskSpace(backup logical.LogicalBackup) error {
	tempDirUsage, err := diskusage.Get(s.tempDir)
	if err != nil {
		return err
	}

	backupSizeBytes := int64(backup.BackupSizeMb * bytesInMegabyte)
	neededBytes := max(backupSizeBytes+int64(float64(backupSizeBytes)*archiveSpaceMargin), minArchiveSpaceBytes)

	if tempDirUsage.FreeBytes < neededBytes {
		return newInsufficientDiskSpaceError(neededBytes, tempDirUsage.FreeBytes)
	}

	return nil
}

// insertRestore inserts the restore with its audit event. A partial unique index refuses a second
// restore of the database, and the reference to the backup a backup deleted meanwhile.
func (s *Service) insertRestore(
	ctx context.Context,
	actor users.Actor,
	backup logical.LogicalBackup,
	backupDatabase databases.Database,
) (Restore, error) {
	startedRestore := Restore{
		ID:                uuid.New(),
		BackupID:          backup.ID,
		DatabaseID:        backup.DatabaseID,
		Status:            StatusInProgress,
		FailMessage:       nil,
		RestoreDurationMs: 0,
		CreatedAt:         s.clock.Now().UTC(),
	}

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.repository.insert(ctx, startedRestore)

		switch {
		case database.IsUniqueViolation(err, inProgressIndexName):
			return ErrRestoreAlreadyRunning
		case database.IsForeignKeyViolation(err):
			return logical.ErrBackupNotFound
		case err != nil:
			return err
		}

		return s.audit.Record(ctx, actor, backupDatabase.WorkspaceID,
			fmt.Sprintf("Database restored for database: %s", backupDatabase.Name))
	})
	if err != nil {
		return Restore{}, err
	}

	return startedRestore, nil
}

func (s *Service) launchRun(
	ctx context.Context,
	startedRestore Restore,
	storedBackup pipeline.StoredBackup,
	openedTarget *pipeline.OpenedRestoreTarget,
) {
	runCtx, cancel, release := runcontext.New(
		s.getAppContext(ctx),
		runcontext.Limits{
			Shutdown:  runcontext.ErrRestoreShutdown,
			Timeout:   runcontext.ErrRestoreTimeout,
			TimeLimit: runcontext.LogicalTimeLimit,
		},
	)
	runCtx = logging.With(runCtx, "restore_id", startedRestore.ID, "backup_id", startedRestore.BackupID)

	s.cancellations.Register(startedRestore.ID, cancel)

	s.runs.Go(runCtx, "logical restore", func(runCtx context.Context) error {
		defer release()
		defer s.cancellations.Unregister(startedRestore.ID)
		defer openedTarget.Close()

		s.runRestore(runCtx, startedRestore, storedBackup, openedTarget)

		return nil
	})
}

func (s *Service) runRestore(
	runCtx context.Context,
	startedRestore Restore,
	storedBackup pipeline.StoredBackup,
	openedTarget *pipeline.OpenedRestoreTarget,
) {
	startedAt := s.clock.Now()

	restoreErr := openedTarget.Restore(runCtx, storedBackup)

	ctx, cancelOutcome := s.shutdownBudget.NewContext(
		runCtx,
		shutdown.ContextSpec{Timeout: 30 * time.Second, Reserve: 0},
	)
	defer cancelOutcome()
	durationMs := s.clock.Now().Sub(startedAt).Milliseconds()

	var err error

	switch {
	case restoreErr == nil:
		_, err = s.repository.complete(ctx, startedRestore.ID, durationMs)

		s.logger.InfoContext(ctx, "logical restore completed", slog.Int64("duration_ms", durationMs))
	case errors.Is(restoreErr, runcontext.ErrRestoreCancelled):
		_, err = s.repository.cancel(ctx, startedRestore.ID, durationMs)

		s.logger.WarnContext(ctx, "logical restore cancelled on request")
	default:
		_, err = s.repository.fail(ctx, startedRestore.ID, restoreErr.Error(), durationMs)

		s.logger.ErrorContext(ctx, "logical restore failed", slog.Any("error", restoreErr))
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to record the outcome of the restore", slog.Any("error", err))
	}
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

// getAuthorizedRestore answers a restore of a workspace the actor is not a member of like a missing one.
func (s *Service) getAuthorizedRestore(
	ctx context.Context,
	actor users.Actor,
	restoreID uuid.UUID,
	action workspaces.Action,
) (Restore, databases.Database, error) {
	foundRestore, isFound, err := s.repository.get(ctx, restoreID)
	if err != nil {
		return Restore{}, databases.Database{}, err
	}

	if !isFound {
		return Restore{}, databases.Database{}, ErrRestoreNotFound
	}

	_, backupDatabase, err := s.logicalBackups.GetAuthorizedBackup(ctx, actor, foundRestore.BackupID, action)
	if errors.Is(err, logical.ErrBackupNotFound) {
		return Restore{}, databases.Database{}, ErrRestoreNotFound
	}

	if err != nil {
		return Restore{}, databases.Database{}, err
	}

	return foundRestore, backupDatabase, nil
}
