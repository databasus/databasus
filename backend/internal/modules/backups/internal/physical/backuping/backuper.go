package backuping

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/workspaces"
)

type Backuper struct {
	liveRunsMutex            sync.Mutex
	liveDatabaseIDs          map[uuid.UUID]struct{}
	lifecycle                *catalog.BackupLifecycle
	databases                databases.Module
	workspaceService         workspaces.Module
	backupRepo               *catalog.BackupRepository
	historyRepo              *catalog.WalHistoryRepository
	configs                  *config.Service
	storages                 storages.Module
	notificationSender       notifiers.Module
	taskCancellationRegistry *background.CancellationRegistry
	masterKey                *crypto.MasterKey
	logger                   *slog.Logger
	clock                    clock.Source
	executionClock           clock.Source
	transactor               *database.Transactor
	clientTools              pgshared.ClientTools
	runs                     *background.Group
	appCtx                   context.Context
}

func (b *Backuper) Start(ctx context.Context) { b.appCtx = ctx }

type registeredBackupRun struct {
	Context  context.Context
	BackupID uuid.UUID
	Finish   func()
}

func (b *Backuper) MakeBackup(ctx context.Context, backupID uuid.UUID, isCallNotifier bool) {
	backup, isFound, err := b.backupRepo.FindByID(context.WithoutCancel(ctx), backupID)
	if err != nil {
		b.logger.ErrorContext(ctx, "failed to look up physical backup", slog.Any("error", err))
		return
	}
	if !isFound || backup.Status != catalog.BackupStatusInProgress {
		return
	}

	ctx = logging.With(ctx, slog.Any("database_id", backup.DatabaseID))
	if err := background.RunRecovering(ctx, func(ctx context.Context) error {
		b.runBackup(ctx, backupRunOptions{Backup: backup, IsCallNotifier: isCallNotifier})
		return nil
	}); err != nil {
		b.finalizeAsError(ctx, backup, catalog.BackupErrorPgBasebackupFailed, err.Error())
	}
}

type backupRunSpec struct {
	BackupID   uuid.UUID
	DatabaseID uuid.UUID
}

func (b *Backuper) registerRun(spec backupRunSpec) (registeredBackupRun, bool) {
	b.liveRunsMutex.Lock()
	defer b.liveRunsMutex.Unlock()

	if _, isRunning := b.liveDatabaseIDs[spec.DatabaseID]; isRunning {
		return registeredBackupRun{}, false
	}

	b.liveDatabaseIDs[spec.DatabaseID] = struct{}{}
	runCtx, cancel, finish := runcontext.New(b.appCtx, runcontext.Limits{Shutdown: runcontext.ErrBackupShutdown})
	runCtx = logging.With(runCtx, slog.Any("backup_id", spec.BackupID))
	b.taskCancellationRegistry.Register(spec.BackupID, cancel)

	var finishOnce sync.Once
	return registeredBackupRun{Context: runCtx, BackupID: spec.BackupID, Finish: func() {
		finishOnce.Do(func() {
			finish()
			b.taskCancellationRegistry.Unregister(spec.BackupID)
			b.liveRunsMutex.Lock()
			defer b.liveRunsMutex.Unlock()
			delete(b.liveDatabaseIDs, spec.DatabaseID)
		})
	}}, true
}

func (b *Backuper) dispatch(run registeredBackupRun, isCallNotifier bool) {
	b.runs.Go(run.Context, "physical_backup", func(ctx context.Context) error {
		defer run.Finish()
		b.MakeBackup(ctx, run.BackupID, isCallNotifier)
		return nil
	})
}

func (b *Backuper) getDatabase(ctx context.Context, databaseID uuid.UUID) (databases.Database, error) {
	listedDatabases, err := b.databases.GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}
	if len(listedDatabases) == 0 {
		return databases.Database{}, errDatabaseNotFound
	}
	return listedDatabases[0], nil
}

func (b *Backuper) openBackupContext(
	ctx context.Context,
	logger *slog.Logger,
	databaseID uuid.UUID,
) (*backupContext, bool) {
	logger = logger.With(slog.Any("database_id", databaseID))

	cfg, err := b.configs.GetPhysicalBackupConfig(ctx, databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch physical backup config", slog.Any("error", err))

		return nil, false
	}

	if cfg.StorageID == nil {
		logger.ErrorContext(ctx, "physical backup config has no storage id")

		return nil, false
	}

	db, err := b.getDatabase(ctx, databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch database by id", slog.Any("error", err))

		return nil, false
	}

	connectionSettings, err := b.databases.GetPostgresqlPhysicalConnectionSettings(ctx, databaseID)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to read the connection settings of the physical database",
			slog.Any("error", err),
		)

		return nil, false
	}

	storage, err := getStorage(ctx, b.storages, *cfg.StorageID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch storage", slog.Any("error", err))

		return nil, false
	}

	var masterKey *crypto.MasterKey
	if cfg.Encryption == config.EncryptionEncrypted {
		masterKey = b.masterKey
	}

	tunnelCtx, cancelTunnelOpen := context.WithTimeout(ctx, bastionOpenTimeout)
	defer cancelTunnelOpen()

	tunnel, err := pgphysical.NewSource(connectionSettings.Engine).OpenTunnel(tunnelCtx, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to open ssh tunnel to the source cluster", slog.Any("error", err))

		return nil, false
	}

	sourceThroughTunnel := connectionSettings
	sourceThroughTunnel.Engine = tunnel.GetSource().GetSettings()

	return &backupContext{
		Config:    cfg,
		Database:  db,
		Source:    sourceThroughTunnel,
		Storage:   storage,
		MasterKey: masterKey,
		tunnel:    tunnel,
	}, true
}

func (b *Backuper) resolveParentManifest(ctx context.Context, backup *catalog.Backup) (ParentManifestRef, error) {
	parent, _, err := b.backupRepo.FindByID(ctx, *backup.ParentBackupID)
	if err != nil {
		return ParentManifestRef{}, fmt.Errorf("look up parent backup: %w", err)
	}
	if parent == nil || parent.ManifestFileName == nil || parent.StopLSN == nil {
		return ParentManifestRef{}, errors.New("parent backup missing or has no manifest_file_name / stop_lsn")
	}
	return ParentManifestRef{
		BackupID: parent.ID, FileName: *parent.ManifestFileName,
		Encryption: parent.Encryption, Salt: derefString(parent.ManifestEncryptionSalt),
		IV: derefString(parent.ManifestEncryptionIV), StopLSN: *parent.StopLSN,
	}, nil
}

func (b *Backuper) getSourceClusterSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	source databases.PostgresqlPhysicalConnectionSettings,
) *float64 {
	probeCtx, cancelProbe := context.WithTimeout(ctx, clusterSizeProbeTimeout)
	defer cancelProbe()

	clusterSizeMb, err := pgphysical.NewSource(source.Engine).GetClusterSizeMb(probeCtx, logger)
	if err != nil {
		logger.WarnContext(ctx, "failed to measure source cluster size", slog.Any("error", err))

		return nil
	}

	return &clusterSizeMb
}

func (b *Backuper) persistResult(
	ctx context.Context,
	backup *catalog.Backup,
	backupResult Result,
	rawSizeMb *float64,
) error {
	if backupResult.FileName == "" {
		backupResult.FileName = derefString(backup.FileName)
	}
	backup.Status = backupResult.Status
	backup.ErrorReason = backupResult.ErrorReason
	backup.FailMessage = nilOrPtr(backupResult.ErrorMessage)

	if backupResult.Status == catalog.BackupStatusCompleted {
		backup.TimelineID = backupResult.TimelineID
		backup.StartLSN = lsnPtr(backupResult.StartLSN)
		backup.StopLSN = lsnPtr(backupResult.StopLSN)
		backup.BackupSizeMb = &backupResult.BackupSizeMb
		backup.RawSizeMb = rawSizeMb
		backup.BackupDurationMs = &backupResult.BackupDurationMs
		backup.Encryption = backupResult.EncryptionAlgo
		backup.EncryptionSalt = nilOrPtr(backupResult.EncryptionSalt)
		backup.EncryptionIV = nilOrPtr(backupResult.EncryptionIV)

		backup.Compression = backupResult.Compression
		backup.ManifestFileName = nilOrPtr(backupResult.ManifestFileName)
		backup.ManifestEncryptionSalt = nilOrPtr(backupResult.ManifestEncryptionSalt)
		backup.ManifestEncryptionIV = nilOrPtr(backupResult.ManifestEncryptionIV)
	}

	completedAt := backupResult.CompletedAt
	if completedAt.IsZero() {
		completedAt = b.clock.Now()
	}
	backup.CompletedAt = &completedAt

	backup.FileName = nilOrPtr(backupResult.FileName)
	if backup.ManifestFileName == nil {
		backup.ManifestFileName = nilOrPtr(backupResult.ManifestFileName)
	}

	var err error
	var isChanged bool
	switch backupResult.Status {
	case catalog.BackupStatusCompleted:
		isChanged, err = b.lifecycle.Complete(ctx, backup, backupResult.Receipts)
	case catalog.BackupStatusChainBroken:
		isChanged, err = b.lifecycle.MarkChainBroken(ctx, backup)
	case catalog.BackupStatusCanceled:
		isChanged, err = b.lifecycle.Cancel(ctx, backup)
	case catalog.BackupStatusError:
		isChanged, err = b.lifecycle.Fail(ctx, backup)
	case catalog.BackupStatusInProgress:
		panic("cannot persist a running backup as terminal")
	}
	if err == nil && !isChanged {
		return errBackupRowSuperseded
	}
	return err
}

func (b *Backuper) isIncrRetryBudgetExhausted(
	ctx context.Context,
	logger *slog.Logger,
	incrBackup *catalog.Backup,
) bool {
	recentBackups, err := b.backupRepo.FindNewestAnyStatusByRootFull(ctx,
		*incrBackup.RootFullBackupID, maxConsecutiveFailedIncrTries)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read the chain's recent incrementals", slog.Any("error", err))

		return false
	}

	priorBackups := slices.DeleteFunc(recentBackups,
		func(backup *catalog.Backup) bool {
			return backup.ID == incrBackup.ID
		})

	if len(priorBackups) != maxConsecutiveFailedIncrTries-1 {
		return false
	}

	return !slices.ContainsFunc(priorBackups,
		func(backup *catalog.Backup) bool {
			return backup.Status != catalog.BackupStatusError
		})
}

func (b *Backuper) finalizeAsError(
	ctx context.Context,
	backup *catalog.Backup,
	reason catalog.BackupErrorReason,
	message string,
) {
	b.finalizeWithStatus(ctx, backup, catalog.BackupStatusError, reason, message)
}

func (b *Backuper) finalizeAsChainBroken(
	ctx context.Context,
	backup *catalog.Backup,
	reason catalog.BackupErrorReason,
	message string,
) {
	b.finalizeWithStatus(ctx, backup, catalog.BackupStatusChainBroken, reason, message)
}

func (b *Backuper) finalizeWithStatus(
	ctx context.Context,
	backup *catalog.Backup,
	status catalog.BackupStatus,
	reason catalog.BackupErrorReason,
	message string,
) {
	backupResult := classifyShutdown(
		ctx,
		Result{
			Status:           status,
			ErrorReason:      &reason,
			ErrorMessage:     message,
			FileName:         derefString(backup.FileName),
			ManifestFileName: derefString(backup.ManifestFileName),
		},
	)

	cleanupCtx := context.WithoutCancel(ctx)
	if err := b.persistResult(cleanupCtx, backup, backupResult, nil); err != nil {
		b.logger.ErrorContext(cleanupCtx, "failed to record physical backup failure", slog.Any("error", err))
		return
	}

	if errors.Is(context.Cause(ctx), runcontext.ErrBackupShutdown) {
		backupConfig, err := b.configs.GetPhysicalBackupConfig(cleanupCtx, backup.DatabaseID)
		if err != nil {
			return
		}
		backupDatabase, err := b.getDatabase(cleanupCtx, backup.DatabaseID)
		if err != nil {
			return
		}
		b.sendBackupNotification(cleanupCtx, backupConfig, backupDatabase, backup, backupResult)
	}
}

func (b *Backuper) sendBackupNotification(
	ctx context.Context,
	cfg config.PhysicalBackupConfig,
	db databases.Database,
	backup *catalog.Backup,
	backupResult Result,
) {
	notificationType, title, message := classifyBackupNotification(
		ctx,
		db,
		backup,
		backupResult,
		b.workspaceService,
	)
	if notificationType == "" {
		return
	}

	if !slices.Contains(cfg.NotificationTriggers, notificationType) {
		return
	}

	notification := notifiers.Notification{
		Category: toNotificationType(notificationType),
		Heading:  title,
		Message:  message,
	}

	b.notificationSender.Notify(ctx, db.NotifierIDs, notification)
}

func toNotificationType(
	backupNotificationType config.NotificationTrigger,
) notifiers.Category {
	if backupNotificationType == config.NotificationTriggerBackupSuccess {
		return notifiers.CategoryBackupSuccess
	}

	return notifiers.CategoryBackupFailed
}

func classifyBackupNotification(
	ctx context.Context,
	db databases.Database,
	backup *catalog.Backup,
	backupResult Result,
	workspaceService workspaces.Module,
) (config.NotificationTrigger, string, string) {
	workspaceName := "unknown"
	if listedWorkspaces, err := workspaceService.GetWorkspaces(
		ctx,
		[]uuid.UUID{db.WorkspaceID},
	); err == nil &&
		len(listedWorkspaces) > 0 {
		workspaceName = listedWorkspaces[0].Name
	}

	return classifyBackupNotificationSpec(
		backupNotificationSpec{
			Database:      db,
			WorkspaceName: workspaceName,
			BackupResult:  backupResult,
			Backup:        backup,
		},
	)
}

func reasonOrEmpty(r *catalog.BackupErrorReason) string {
	if r == nil {
		return ""
	}

	return string(*r)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func nilOrPtr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func lsnPtr(v walmath.LSN) *walmath.LSN {
	out := v

	return &out
}

func classifyShutdown(ctx context.Context, backupResult Result) Result {
	if backupResult.Status != catalog.BackupStatusCompleted &&
		errors.Is(context.Cause(ctx), runcontext.ErrBackupShutdown) {
		backupResult.Status = catalog.BackupStatusError
		backupResult.ErrorReason = new(catalog.BackupErrorApplicationRestart)
		backupResult.ErrorMessage = runcontext.ErrBackupShutdown.Error()
	}

	return backupResult
}
