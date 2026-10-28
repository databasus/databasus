package backuping

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

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

const metadataSuffix = ".metadata"

type Backuper struct {
	databases                databases.Module
	workspaceService         workspaces.Module
	fullRepo                 *catalog.FullBackupRepository
	incrRepo                 *catalog.IncrementalBackupRepository
	inFlightRepo             *catalog.InFlightBackupRepository
	historyRepo              *catalog.WalHistoryRepository
	configs                  *config.Service
	storages                 storages.Module
	notificationSender       notifiers.Module
	taskCancellationRegistry *background.CancellationRegistry
	masterKey                *crypto.MasterKey
	logger                   *slog.Logger
	fullExecutor             *FullExecutor
	incrExecutor             *IncrementalExecutor
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
	logger := b.logger

	fullBackup, _, err := b.fullRepo.FindByID(context.WithoutCancel(ctx), backupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to look up full backup row", slog.Any("error", err))
		return
	}

	if fullBackup != nil {
		ctx = logging.With(ctx, slog.Any("database_id", fullBackup.DatabaseID))
		if err := background.RunRecovering(
			ctx,
			func(ctx context.Context) error { b.runFullBackup(ctx, logger, fullBackup, isCallNotifier); return nil },
		); err != nil {
			b.finalizeFullAsError(ctx, fullBackup, catalog.BackupErrorPgBasebackupFailed, err.Error())
		}
		return
	}

	incrBackup, _, err := b.incrRepo.FindByID(context.WithoutCancel(ctx), backupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to look up incremental backup row", slog.Any("error", err))

		return
	}

	if incrBackup != nil {
		ctx = logging.With(ctx, slog.Any("database_id", incrBackup.DatabaseID))
		if err := background.RunRecovering(ctx, func(ctx context.Context) error {
			b.runIncrementalBackup(ctx, logger, incrBackup, isCallNotifier)
			return nil
		}); err != nil {
			b.finalizeIncrAsError(ctx, incrBackup, catalog.BackupErrorPgBasebackupFailed, err.Error())
		}
		return
	}

	logger.WarnContext(ctx, "backup not found in either typed table; ignoring assignment")
}

func (b *Backuper) registerRun(backupID uuid.UUID) registeredBackupRun {
	runCtx, cancel, finish := runcontext.New(b.appCtx, runcontext.Limits{Shutdown: runcontext.ErrBackupShutdown})
	runCtx = logging.With(runCtx, slog.Any("backup_id", backupID))
	b.taskCancellationRegistry.Register(backupID, cancel)
	return registeredBackupRun{Context: runCtx, BackupID: backupID, Finish: func() {
		finish()
		b.taskCancellationRegistry.Unregister(backupID)
	}}
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

func (b *Backuper) runFullBackup(
	ctx context.Context,
	logger *slog.Logger,
	fullBackup *catalog.FullBackup,
	isCallNotifier bool,
) {
	backupCtx, ok := b.openBackupContext(ctx, logger, fullBackup.DatabaseID)
	if !ok {
		b.finalizeFullAsError(ctx, fullBackup, catalog.BackupErrorPgBasebackupFailed,
			"failed to open backup context")

		return
	}

	defer backupCtx.Close()

	executionCtx := ctx

	rawSizeMb := b.getSourceClusterSizeMb(executionCtx, logger, backupCtx.Source)

	fullBackupSpec := FullBackupSpec{
		CommonBackupSpec: CommonBackupSpec{
			ExecutionClock: b.executionClock,
			Clock:          b.clock, Transactor: b.transactor,
			ClientTools:  b.clientTools,
			SourceDB:     backupCtx.Source,
			DatabaseName: backupCtx.Database.Name,
			StorageID:    backupCtx.Storage.ID,
			FileStore:    b.storages,
			Encryption:   catalog.BackupEncryption(backupCtx.Config.Encryption),
			MasterKey:    backupCtx.MasterKey,
			FullRepo:     b.fullRepo,
			HistoryRepo:  b.historyRepo,
			Logger:       logger,
			ProgressListener: func(completedMb float64, elapsedMs int64) {
				if err := b.fullRepo.UpdateProgress(ctx, fullBackup.ID, completedMb, elapsedMs); err != nil {
					logger.ErrorContext(ctx, "failed to update full backup progress", slog.Any("error", err))
				}
			},
		},
		Backup: fullBackup,
	}

	sourceSize := "unknown"
	if rawSizeMb != nil {
		sourceSize = fmt.Sprintf("%.1f MB", *rawSizeMb)
	}

	logger.InfoContext(
		ctx,
		"physical full backup started",
		slog.Any("database_name", backupCtx.Database.Name),
		slog.Any("source_size", sourceSize),
		slog.Any("storage_id", backupCtx.Storage.ID),
		slog.Any("encryption", backupCtx.Config.Encryption),
	)

	backupResult, err := b.fullExecutor.Execute(executionCtx, fullBackupSpec)
	if err != nil {
		logger.ErrorContext(ctx, "full executor returned error", slog.Any("error", err))

		b.finalizeFullAsError(ctx, fullBackup, catalog.BackupErrorPgBasebackupFailed, err.Error())

		return
	}

	backupResult = classifyShutdown(ctx, backupResult)
	ctx = context.WithoutCancel(ctx)

	if backupResult.Status != catalog.BackupStatusCompleted {
		logger.WarnContext(
			ctx,
			"full executor returned non-completed result",
			slog.Any("status", backupResult.Status),
			slog.Any("reason", reasonOrEmpty(backupResult.ErrorReason)),
			slog.Any("message", backupResult.ErrorMessage),
		)
	}

	if err := b.persistFullResult(ctx, fullBackup, backupResult, rawSizeMb); err != nil {
		logger.ErrorContext(ctx, "failed to persist full result", slog.Any("error", err))

		return
	}

	logger.InfoContext(
		ctx,
		"physical full backup finished",
		slog.Any("status", backupResult.Status),
		slog.Any("size_mb", backupResult.BackupSizeMb),
		slog.Any("duration_ms", backupResult.BackupDurationMs),
		slog.Any("timeline_id", backupResult.TimelineID),
		slog.Any("start_lsn", backupResult.StartLSN),
		slog.Any("stop_lsn", backupResult.StopLSN),
	)

	if isCallNotifier {
		b.sendFullBackupNotification(ctx, backupCtx.Config, backupCtx.Database, fullBackup, backupResult)
	}
}

func (b *Backuper) runIncrementalBackup(
	ctx context.Context,
	logger *slog.Logger,
	incrBackup *catalog.IncrementalBackup,
	isCallNotifier bool,
) {
	backupCtx, ok := b.openBackupContext(ctx, logger, incrBackup.DatabaseID)
	if !ok {
		b.finalizeIncrAsError(ctx, incrBackup, catalog.BackupErrorPgBasebackupFailed,
			"failed to open backup context")

		return
	}

	defer backupCtx.Close()

	parentRef, err := b.resolveParentManifest(ctx, incrBackup)
	if err != nil {
		logger.ErrorContext(ctx, "failed to resolve parent manifest", slog.Any("error", err))

		b.finalizeIncrAsChainBroken(ctx, incrBackup,
			catalog.BackupErrorParentManifestMissing, err.Error())

		return
	}

	storageReader, err := b.storages.GetStorageReader(ctx, backupCtx.Storage.ID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to open the storage of the parent manifest", slog.Any("error", err))

		b.finalizeIncrAsError(ctx, incrBackup, catalog.BackupErrorPgBasebackupFailed,
			"failed to open the storage of the parent manifest")

		return
	}

	executionCtx := ctx

	var incrementalCadence time.Duration
	if backupCtx.Config.IncrementalBackupSchedule != nil {
		incrementalCadence = backupCtx.Config.IncrementalBackupSchedule.GetApproxPeriod()
	}

	incrBackupSpec := IncrementalBackupSpec{
		CommonBackupSpec: CommonBackupSpec{
			ExecutionClock: b.executionClock,
			Clock:          b.clock, Transactor: b.transactor,
			ClientTools:  b.clientTools,
			SourceDB:     backupCtx.Source,
			DatabaseName: backupCtx.Database.Name,
			StorageID:    backupCtx.Storage.ID,
			FileStore:    b.storages,
			Encryption:   catalog.BackupEncryption(backupCtx.Config.Encryption),
			MasterKey:    backupCtx.MasterKey,
			FullRepo:     b.fullRepo,
			HistoryRepo:  b.historyRepo,
			Logger:       logger,
			ProgressListener: func(completedMb float64, elapsedMs int64) {
				if err := b.incrRepo.UpdateProgress(ctx, incrBackup.ID, completedMb, elapsedMs); err != nil {
					logger.ErrorContext(ctx, "failed to update incremental backup progress", slog.Any("error", err))
				}
			},
		},
		Backup:             incrBackup,
		ParentManifest:     parentRef,
		StorageReader:      storageReader,
		IncrRepo:           b.incrRepo,
		IncrementalCadence: incrementalCadence,
	}

	logger.InfoContext(
		ctx,
		"physical incremental backup started",
		slog.Any("database_name", backupCtx.Database.Name),
		slog.Any("parent_stop_lsn", parentRef.StopLSN),
		slog.Any("parent_backup_id", parentRef.BackupID),
		slog.Any("storage_id", backupCtx.Storage.ID),
	)

	backupResult, err := b.incrExecutor.Execute(executionCtx, incrBackupSpec)
	if err != nil {
		logger.ErrorContext(ctx, "incremental executor returned error", slog.Any("error", err))

		b.finalizeIncrAsError(ctx, incrBackup, catalog.BackupErrorPgBasebackupFailed, err.Error())

		return
	}

	if !errors.Is(context.Cause(ctx), runcontext.ErrBackupShutdown) &&
		backupResult.Status == catalog.BackupStatusError &&
		b.isIncrRetryBudgetExhausted(ctx, logger, incrBackup) {
		backupResult.Status = catalog.BackupStatusChainBroken
	}

	backupResult = classifyShutdown(ctx, backupResult)
	ctx = context.WithoutCancel(ctx)

	if backupResult.Status != catalog.BackupStatusCompleted {
		logger.WarnContext(
			ctx,
			"incremental executor returned non-completed result",
			slog.Any("status", backupResult.Status),
			slog.Any("reason", reasonOrEmpty(backupResult.ErrorReason)),
			slog.Any("message", backupResult.ErrorMessage),
		)
	}

	if err := b.persistIncrResult(ctx, incrBackup, backupResult); err != nil {
		logger.ErrorContext(ctx, "failed to persist incremental result", slog.Any("error", err))

		return
	}

	logger.InfoContext(
		ctx,
		"physical incremental backup finished",
		slog.Any("status", backupResult.Status),
		slog.Any("size_mb", backupResult.BackupSizeMb),
		slog.Any("duration_ms", backupResult.BackupDurationMs),
		slog.Any("timeline_id", backupResult.TimelineID),
		slog.Any("start_lsn", backupResult.StartLSN),
		slog.Any("stop_lsn", backupResult.StopLSN),
	)

	if isCallNotifier {
		b.sendIncrBackupNotification(ctx, backupCtx.Config, backupCtx.Database, incrBackup, backupResult)
	}
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

func (b *Backuper) resolveParentManifest(
	ctx context.Context,
	incrBackup *catalog.IncrementalBackup,
) (ParentManifestRef, error) {
	if incrBackup.ParentIncrementalBackupID != nil {
		parent, _, lookupErr := b.incrRepo.FindByID(ctx, *incrBackup.ParentIncrementalBackupID)
		if lookupErr != nil {
			return ParentManifestRef{}, fmt.Errorf("look up parent incr: %w", lookupErr)
		}

		if parent == nil || parent.ManifestFileName == nil || parent.StopLSN == nil {
			return ParentManifestRef{}, errors.New(
				"parent incremental row missing or has no manifest_file_name / stop_lsn",
			)
		}

		return ParentManifestRef{
			BackupID:   parent.ID,
			FileName:   *parent.ManifestFileName,
			Encryption: parent.Encryption,
			Salt:       derefString(parent.ManifestEncryptionSalt),
			IV:         derefString(parent.ManifestEncryptionIV),
			StopLSN:    *parent.StopLSN,
		}, nil
	}

	parent, _, lookupErr := b.fullRepo.FindByID(ctx, incrBackup.RootFullBackupID)
	if lookupErr != nil {
		return ParentManifestRef{}, fmt.Errorf("look up root full: %w", lookupErr)
	}

	if parent == nil || parent.ManifestFileName == nil || parent.StopLSN == nil {
		return ParentManifestRef{}, errors.New(
			"root full row missing or has no manifest_file_name / stop_lsn")
	}

	return ParentManifestRef{
		BackupID:   parent.ID,
		FileName:   *parent.ManifestFileName,
		Encryption: parent.Encryption,
		Salt:       derefString(parent.ManifestEncryptionSalt),
		IV:         derefString(parent.ManifestEncryptionIV),
		StopLSN:    *parent.StopLSN,
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

func (b *Backuper) persistFullResult(
	ctx context.Context,
	fullBackup *catalog.FullBackup,
	backupResult Result,
	rawSizeMb *float64,
) error {
	if backupResult.FileName == "" {
		backupResult.FileName = derefString(fullBackup.FileName)
	}
	fullBackup.Status = backupResult.Status
	fullBackup.ErrorReason = backupResult.ErrorReason
	fullBackup.FailMessage = nilOrPtr(backupResult.ErrorMessage)

	if backupResult.Status == catalog.BackupStatusCompleted {
		fullBackup.TimelineID = backupResult.TimelineID
		fullBackup.StartLSN = lsnPtr(backupResult.StartLSN)
		fullBackup.StopLSN = lsnPtr(backupResult.StopLSN)
		fullBackup.BackupSizeMb = &backupResult.BackupSizeMb
		fullBackup.RawSizeMb = rawSizeMb
		fullBackup.BackupDurationMs = &backupResult.BackupDurationMs
		fullBackup.Encryption = backupResult.EncryptionAlgo
		fullBackup.EncryptionSalt = nilOrPtr(backupResult.EncryptionSalt)
		fullBackup.EncryptionIV = nilOrPtr(backupResult.EncryptionIV)

		fullBackup.Compression = backupResult.Compression
		fullBackup.ManifestFileName = nilOrPtr(backupResult.ManifestFileName)
		fullBackup.ManifestEncryptionSalt = nilOrPtr(backupResult.ManifestEncryptionSalt)
		fullBackup.ManifestEncryptionIV = nilOrPtr(backupResult.ManifestEncryptionIV)

	}

	completedAt := backupResult.CompletedAt
	if completedAt.IsZero() {
		completedAt = b.clock.Now()
	}
	fullBackup.CompletedAt = &completedAt

	return b.saveTerminalResultIfInProgress(ctx, terminalPersistSpec{
		DatabaseID:      fullBackup.DatabaseID,
		BackupID:        fullBackup.ID,
		BackupCreatedAt: fullBackup.CreatedAt,
		StorageID:       fullBackup.StorageID,
		Result:          backupResult,
		LoadStatus: func(ctx context.Context) (catalog.BackupStatus, bool, error) {
			return b.fullRepo.GetStatusForUpdate(ctx, fullBackup.ID)
		},
		Save: func(ctx context.Context) error { return b.fullRepo.UpdateTerminalResult(ctx, fullBackup) },
	})
}

func (b *Backuper) persistIncrResult(
	ctx context.Context,
	incrBackup *catalog.IncrementalBackup,
	backupResult Result,
) error {
	if backupResult.FileName == "" {
		backupResult.FileName = derefString(incrBackup.FileName)
	}
	incrBackup.Status = backupResult.Status
	incrBackup.ErrorReason = backupResult.ErrorReason
	incrBackup.FailMessage = nilOrPtr(backupResult.ErrorMessage)

	if backupResult.Status == catalog.BackupStatusCompleted {
		incrBackup.TimelineID = backupResult.TimelineID
		incrBackup.StartLSN = lsnPtr(backupResult.StartLSN)
		incrBackup.StopLSN = lsnPtr(backupResult.StopLSN)
		incrBackup.BackupSizeMb = &backupResult.BackupSizeMb
		incrBackup.BackupDurationMs = &backupResult.BackupDurationMs
		incrBackup.Encryption = backupResult.EncryptionAlgo
		incrBackup.EncryptionSalt = nilOrPtr(backupResult.EncryptionSalt)
		incrBackup.EncryptionIV = nilOrPtr(backupResult.EncryptionIV)

		incrBackup.Compression = backupResult.Compression
		incrBackup.ManifestFileName = nilOrPtr(backupResult.ManifestFileName)
		incrBackup.ManifestEncryptionSalt = nilOrPtr(backupResult.ManifestEncryptionSalt)
		incrBackup.ManifestEncryptionIV = nilOrPtr(backupResult.ManifestEncryptionIV)

	}

	completedAt := backupResult.CompletedAt
	if completedAt.IsZero() {
		completedAt = b.clock.Now()
	}
	incrBackup.CompletedAt = &completedAt

	return b.saveTerminalResultIfInProgress(ctx, terminalPersistSpec{
		DatabaseID:      incrBackup.DatabaseID,
		BackupID:        incrBackup.ID,
		BackupCreatedAt: incrBackup.CreatedAt,
		StorageID:       incrBackup.StorageID,
		Result:          backupResult,
		LoadStatus: func(ctx context.Context) (catalog.BackupStatus, bool, error) {
			return b.incrRepo.GetStatusForUpdate(ctx, incrBackup.ID)
		},
		Save: func(ctx context.Context) error { return b.incrRepo.UpdateTerminalResult(ctx, incrBackup) },
	})
}

func (b *Backuper) isIncrRetryBudgetExhausted(
	ctx context.Context,
	logger *slog.Logger,
	incrBackup *catalog.IncrementalBackup,
) bool {
	recentBackups, err := b.incrRepo.FindNewestAnyStatusByRootFull(ctx,
		incrBackup.RootFullBackupID, maxConsecutiveFailedIncrTries)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read the chain's recent incrementals", slog.Any("error", err))

		return false
	}

	priorBackups := slices.DeleteFunc(recentBackups,
		func(backup *catalog.IncrementalBackup) bool {
			return backup.ID == incrBackup.ID
		})

	if len(priorBackups) != maxConsecutiveFailedIncrTries-1 {
		return false
	}

	return !slices.ContainsFunc(priorBackups,
		func(backup *catalog.IncrementalBackup) bool {
			return backup.Status != catalog.BackupStatusError
		})
}

type terminalPersistSpec struct {
	DatabaseID uuid.UUID
	BackupID   uuid.UUID

	BackupCreatedAt time.Time
	StorageID       uuid.UUID
	Result          Result
	LoadStatus      func(ctx context.Context) (catalog.BackupStatus, bool, error)
	Save            func(ctx context.Context) error
}

func (b *Backuper) saveTerminalResultIfInProgress(ctx context.Context, spec terminalPersistSpec) error {
	err := b.transactor.Transaction(ctx, func(ctx context.Context) error {
		isCompleted := spec.Result.Status == catalog.BackupStatusCompleted
		if isCompleted {
			err := b.databases.RecordBackupSuccess(ctx, spec.DatabaseID, spec.BackupCreatedAt)
			if err != nil {
				return err
			}
		}

		status, isFound, err := spec.LoadStatus(ctx)
		if err != nil {
			return err
		}

		if !isFound || status != catalog.BackupStatusInProgress {
			return errBackupRowSuperseded
		}

		if isCompleted {
			err = b.storages.ConfirmFileWrites(ctx, spec.Result.Receipts)
		} else {
			err = b.storages.RequestFileDeletions(ctx, b.attemptFileReferences(spec))
		}

		if err != nil {
			return err
		}

		if err := spec.Save(ctx); err != nil {
			return err
		}

		return b.inFlightRepo.ReleaseOwned(ctx, spec.DatabaseID, spec.BackupID)
	})
	if !errors.Is(err, errBackupRowSuperseded) {
		return err
	}

	b.logger.WarnContext(
		ctx,
		"backup row no longer in progress; skipping terminal persist",
		slog.Any("backup_id", spec.BackupID),
	)

	return b.transactor.Transaction(ctx, func(ctx context.Context) error {
		return b.storages.RequestFileDeletions(ctx, b.attemptFileReferences(spec))
	})
}

func (b *Backuper) attemptFileReferences(
	spec terminalPersistSpec,
) []storages.StoredFileReference {
	if spec.Result.FileName == "" {
		return nil
	}

	references := []storages.StoredFileReference{
		{StorageID: spec.StorageID, FileName: spec.Result.FileName},
		{StorageID: spec.StorageID, FileName: spec.Result.FileName + metadataSuffix},
	}

	manifestFileName := spec.Result.ManifestFileName
	if manifestFileName == "" {
		manifestFileName = spec.Result.FileName + catalog.ManifestSuffix
	}
	references = append(references, storages.StoredFileReference{StorageID: spec.StorageID, FileName: manifestFileName})

	return references
}

func (b *Backuper) finalizeFullAsError(
	ctx context.Context,
	fullBackup *catalog.FullBackup,
	reason catalog.BackupErrorReason,
	message string,
) {
	backupResult := classifyShutdown(
		ctx,
		Result{
			Status:           catalog.BackupStatusError,
			ErrorReason:      &reason,
			ErrorMessage:     message,
			FileName:         derefString(fullBackup.FileName),
			ManifestFileName: derefString(fullBackup.ManifestFileName),
		},
	)
	cleanupCtx := context.WithoutCancel(ctx)
	if err := b.persistFullResult(cleanupCtx, fullBackup, backupResult, nil); err != nil {
		b.logger.ErrorContext(cleanupCtx, "failed to record full backup failure", slog.Any("error", err))
		return
	}
	if errors.Is(context.Cause(ctx), runcontext.ErrBackupShutdown) {
		backupConfig, err := b.configs.GetPhysicalBackupConfig(cleanupCtx, fullBackup.DatabaseID)
		if err != nil {
			b.logger.ErrorContext(cleanupCtx, "failed to load shutdown notification config", slog.Any("error", err))
			return
		}
		backupDatabase, err := b.getDatabase(cleanupCtx, fullBackup.DatabaseID)
		if err != nil {
			return
		}
		b.sendFullBackupNotification(cleanupCtx, backupConfig, backupDatabase, fullBackup, backupResult)
	}
}

func (b *Backuper) finalizeIncrAsError(
	ctx context.Context,
	incrBackup *catalog.IncrementalBackup,
	reason catalog.BackupErrorReason,
	message string,
) {
	b.finalizeIncrWithStatus(ctx, incrBackup, catalog.BackupStatusError, reason, message)
}

func (b *Backuper) finalizeIncrAsChainBroken(
	ctx context.Context,
	incrBackup *catalog.IncrementalBackup,
	reason catalog.BackupErrorReason,
	message string,
) {
	b.finalizeIncrWithStatus(ctx, incrBackup, catalog.BackupStatusChainBroken, reason, message)
}

func (b *Backuper) finalizeIncrWithStatus(
	ctx context.Context,
	incrBackup *catalog.IncrementalBackup,
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
			FileName:         derefString(incrBackup.FileName),
			ManifestFileName: derefString(incrBackup.ManifestFileName),
		},
	)
	cleanupCtx := context.WithoutCancel(ctx)
	if err := b.persistIncrResult(cleanupCtx, incrBackup, backupResult); err != nil {
		b.logger.ErrorContext(cleanupCtx, "failed to record incremental backup failure", slog.Any("error", err))
		return
	}
	if errors.Is(context.Cause(ctx), runcontext.ErrBackupShutdown) {
		backupConfig, err := b.configs.GetPhysicalBackupConfig(cleanupCtx, incrBackup.DatabaseID)
		if err != nil {
			return
		}
		backupDatabase, err := b.getDatabase(cleanupCtx, incrBackup.DatabaseID)
		if err != nil {
			return
		}
		b.sendIncrBackupNotification(cleanupCtx, backupConfig, backupDatabase, incrBackup, backupResult)
	}
}

func (b *Backuper) sendFullBackupNotification(
	ctx context.Context,
	cfg config.PhysicalBackupConfig,
	db databases.Database,
	fullBackup *catalog.FullBackup,
	backupResult Result,
) {
	notificationType, title, message := classifyFullBackupNotification(
		ctx,
		db,
		fullBackup,
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

func (b *Backuper) sendIncrBackupNotification(
	ctx context.Context,
	cfg config.PhysicalBackupConfig,
	db databases.Database,
	incrBackup *catalog.IncrementalBackup,
	backupResult Result,
) {
	notificationType, title, message := classifyIncrBackupNotification(
		ctx,
		db,
		incrBackup,
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

func classifyFullBackupNotification(
	ctx context.Context,
	db databases.Database,
	fullBackup *catalog.FullBackup,
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

	return classifyFullNotification(
		backupNotificationSpec{
			Database:      db,
			WorkspaceName: workspaceName,
			BackupResult:  backupResult,
			FullBackup:    fullBackup,
		},
	)
}

func classifyIncrBackupNotification(
	ctx context.Context,
	db databases.Database,
	incrBackup *catalog.IncrementalBackup,
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

	return classifyIncrNotification(
		backupNotificationSpec{
			Database:          db,
			WorkspaceName:     workspaceName,
			BackupResult:      backupResult,
			IncrementalBackup: incrBackup,
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
	if errors.Is(context.Cause(ctx), runcontext.ErrBackupShutdown) {
		backupResult.Status = catalog.BackupStatusError
		backupResult.ErrorReason = new(catalog.BackupErrorApplicationRestart)
		backupResult.ErrorMessage = runcontext.ErrBackupShutdown.Error()
	}
	return backupResult
}
