package backuping_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/storage"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

// PhysicalBackuper drives a FULL or INCR through the postgresql executor.
// The scheduler invokes MakeBackup directly in a goroutine.
// Every stored artifact carries a sidecar under the same name plus this suffix.
const metadataSuffix = ".metadata"

type PhysicalBackuper struct {
	databaseService          *databases.DatabaseService
	fieldEncryptor           util_encryption.FieldEncryptor
	workspaceService         *workspaces_services.WorkspaceService
	fullRepo                 *physical_repositories.PhysicalFullBackupRepository
	incrRepo                 *physical_repositories.PhysicalIncrementalBackupRepository
	inFlightRepo             *physical_repositories.PhysicalInFlightBackupRepository
	historyRepo              *physical_repositories.PhysicalWalHistoryRepository
	backupConfigService      *backups_config_physical.BackupConfigService
	storageService           *storages.StorageService
	fileStore                *storage_files.Store
	notificationSender       NotificationSender
	taskCancellationRegistry *tasks_cancellation.Registry
	masterKey                *crypto.MasterKey
	logger                   *slog.Logger
	fullExecutor             FullBackupExecutor
	incrExecutor             IncrementalBackupExecutor
}

func (b *PhysicalBackuper) MakeBackup(ctx context.Context, backupID uuid.UUID, isCallNotifier bool) {
	logger := b.logger.With("backup_id", backupID)

	fullBackup, err := b.fullRepo.FindByID(backupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to look up full backup row", "error", err)
		return
	}

	if fullBackup != nil {
		b.runFullBackup(ctx, logger, fullBackup, isCallNotifier)
		return
	}

	incrBackup, err := b.incrRepo.FindByID(backupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to look up incremental backup row", "error", err)

		return
	}

	if incrBackup != nil {
		b.runIncrementalBackup(ctx, logger, incrBackup, isCallNotifier)
		return
	}

	logger.WarnContext(ctx, "backup not found in either typed table; ignoring assignment")
}

func (b *PhysicalBackuper) runFullBackup(
	ctx context.Context,
	logger *slog.Logger,
	fullBackup *physical_models.PhysicalFullBackup,
	isCallNotifier bool,
) {
	backupCtx, ok := b.openBackupContext(ctx, logger, fullBackup.DatabaseID)
	if !ok {
		b.finalizeFullAsError(fullBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed,
			"failed to open backup context")

		return
	}

	defer backupCtx.Close()

	// Detached from the caller so a finished HTTP request cannot cancel a running backup; the
	// caller's ctx stays in use for logging, which is what carries request_id and user_id.
	executionCtx, cancel := context.WithCancel(context.Background())
	b.taskCancellationRegistry.RegisterTask(fullBackup.ID, cancel)
	defer b.taskCancellationRegistry.UnregisterTask(fullBackup.ID)

	rawSizeMb := b.getSourceClusterSizeMb(executionCtx, logger, backupCtx.Database)

	fullBackupSpec := postgresql_executor.FullBackupSpec{
		CommonBackupSpec: postgresql_executor.CommonBackupSpec{
			SourceDB:       backupCtx.Database.PostgresqlPhysical,
			DatabaseName:   backupCtx.Database.Name,
			StorageID:      backupCtx.Storage.ID,
			Storage:        backupCtx.Storage,
			FileStore:      b.fileStore,
			Encryption:     backupCtx.Config.Encryption,
			MasterKey:      backupCtx.MasterKey,
			FieldEncryptor: b.fieldEncryptor,
			FullRepo:       b.fullRepo,
			HistoryRepo:    b.historyRepo,
			Logger:         logger,
			ProgressListener: func(completedMb float64, elapsedMs int64) {
				if err := b.fullRepo.UpdateProgress(fullBackup.ID, completedMb, elapsedMs); err != nil {
					logger.ErrorContext(ctx, "failed to update full backup progress", "error", err)
				}
			},
		},
		Backup: fullBackup,
	}

	// The source-size probe is best effort (it warns and returns nil when the cluster is unreachable),
	// so the started line must not depend on it.
	sourceSize := "unknown"
	if rawSizeMb != nil {
		sourceSize = fmt.Sprintf("%.1f MB", *rawSizeMb)
	}

	logger.InfoContext(ctx, fmt.Sprintf("physical full backup started: database %q, %s source",
		backupCtx.Database.Name, sourceSize),
		"storage_id", backupCtx.Storage.ID, "encryption", backupCtx.Config.Encryption)

	backupResult, err := b.fullExecutor.Execute(executionCtx, fullBackupSpec)
	if err != nil {
		logger.ErrorContext(ctx, "full executor returned error", "error", err)

		b.finalizeFullAsError(fullBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed, err.Error())

		return
	}

	if backupResult.Status != physical_enums.PhysicalBackupStatusCompleted {
		logger.WarnContext(ctx, "full executor returned non-COMPLETED result",
			"status", backupResult.Status,
			"reason", reasonOrEmpty(backupResult.ErrorReason),
			"message", backupResult.ErrorMessage)
	}

	if err := b.persistFullResult(ctx, fullBackup, backupResult, rawSizeMb); err != nil {
		logger.ErrorContext(ctx, "failed to persist full result", "error", err)

		return
	}

	logger.InfoContext(ctx, fmt.Sprintf(
		"physical full backup finished: %s, %.1f MB in %d ms, timeline %d, lsn %s -> %s",
		backupResult.Status, backupResult.BackupSizeMb, backupResult.BackupDurationMs,
		backupResult.TimelineID, backupResult.StartLSN, backupResult.StopLSN))

	if isCallNotifier {
		b.sendFullBackupNotification(ctx, backupCtx.Config, backupCtx.Database, fullBackup, backupResult)
	}
}

func (b *PhysicalBackuper) runIncrementalBackup(
	ctx context.Context,
	logger *slog.Logger,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	isCallNotifier bool,
) {
	backupCtx, ok := b.openBackupContext(ctx, logger, incrBackup.DatabaseID)
	if !ok {
		b.finalizeIncrAsError(incrBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed,
			"failed to open backup context")

		return
	}

	defer backupCtx.Close()

	parentRef, err := b.resolveParentManifest(incrBackup)
	if err != nil {
		logger.ErrorContext(ctx, "failed to resolve parent manifest", "error", err)

		b.finalizeIncrAsChainBroken(incrBackup,
			physical_enums.PhysicalBackupErrorParentManifestMissing, err.Error())

		return
	}

	executionCtx, cancel := context.WithCancel(context.Background())
	b.taskCancellationRegistry.RegisterTask(incrBackup.ID, cancel)
	defer b.taskCancellationRegistry.UnregisterTask(incrBackup.ID)

	incrBackupSpec := postgresql_executor.IncrementalBackupSpec{
		CommonBackupSpec: postgresql_executor.CommonBackupSpec{
			SourceDB:       backupCtx.Database.PostgresqlPhysical,
			DatabaseName:   backupCtx.Database.Name,
			StorageID:      backupCtx.Storage.ID,
			Storage:        backupCtx.Storage,
			FileStore:      b.fileStore,
			Encryption:     backupCtx.Config.Encryption,
			MasterKey:      backupCtx.MasterKey,
			FieldEncryptor: b.fieldEncryptor,
			FullRepo:       b.fullRepo,
			HistoryRepo:    b.historyRepo,
			Logger:         logger,
			ProgressListener: func(completedMb float64, elapsedMs int64) {
				if err := b.incrRepo.UpdateProgress(incrBackup.ID, completedMb, elapsedMs); err != nil {
					logger.ErrorContext(ctx, "failed to update incremental backup progress", "error", err)
				}
			},
		},
		Backup:             incrBackup,
		ParentManifest:     parentRef,
		IncrRepo:           b.incrRepo,
		IncrementalCadence: backupCtx.Config.IncrementalBackupInterval.ApproxPeriod(),
	}

	logger.InfoContext(ctx, fmt.Sprintf("physical incremental backup started: database %q, parent lsn %s",
		backupCtx.Database.Name, parentRef.StopLSN),
		"parent_backup_id", parentRef.BackupID, "storage_id", backupCtx.Storage.ID)

	backupResult, err := b.incrExecutor.Execute(executionCtx, incrBackupSpec)
	if err != nil {
		logger.ErrorContext(ctx, "incremental executor returned error", "error", err)

		b.finalizeIncrAsError(incrBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed, err.Error())

		return
	}

	if backupResult.Status == physical_enums.PhysicalBackupStatusError &&
		b.isIncrRetryBudgetExhausted(ctx, logger, incrBackup) {
		backupResult.Status = physical_enums.PhysicalBackupStatusChainBroken
	}

	if backupResult.Status != physical_enums.PhysicalBackupStatusCompleted {
		logger.WarnContext(ctx, "incremental executor returned non-COMPLETED result",
			"status", backupResult.Status,
			"reason", reasonOrEmpty(backupResult.ErrorReason),
			"message", backupResult.ErrorMessage)
	}

	if err := b.persistIncrResult(ctx, incrBackup, backupResult); err != nil {
		logger.ErrorContext(ctx, "failed to persist incremental result", "error", err)

		return
	}

	logger.InfoContext(ctx, fmt.Sprintf(
		"physical incremental backup finished: %s, %.1f MB in %d ms, timeline %d, lsn %s -> %s",
		backupResult.Status, backupResult.BackupSizeMb, backupResult.BackupDurationMs,
		backupResult.TimelineID, backupResult.StartLSN, backupResult.StopLSN))

	if isCallNotifier {
		b.sendIncrBackupNotification(ctx, backupCtx.Config, backupCtx.Database, incrBackup, backupResult)
	}
}

func (b *PhysicalBackuper) openBackupContext(
	ctx context.Context,
	logger *slog.Logger,
	databaseID uuid.UUID,
) (*backupContext, bool) {
	logger = logger.With("database_id", databaseID)

	cfg, err := b.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch physical backup config", "error", err)

		return nil, false
	}

	if cfg.StorageID == nil {
		logger.ErrorContext(ctx, "physical backup config has no storage id")

		return nil, false
	}

	db, err := b.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch database by id", "error", err)

		return nil, false
	}

	if db.PostgresqlPhysical == nil {
		logger.ErrorContext(ctx, "database is not a physical postgres database")

		return nil, false
	}

	storage, err := b.storageService.GetStorageByID(ctx, *cfg.StorageID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch storage", "error", err)

		return nil, false
	}

	masterKey := ""
	if cfg.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		key := b.masterKey.GetSecret()

		masterKey = key
	}

	// Opened last, because a bail-out after this point would strand a live SSH session and its
	// listener: the caller only gets something to Close once the context is fully built.
	tunnelCtx, cancelTunnelOpen := context.WithTimeout(context.Background(), bastionOpenTimeout)
	defer cancelTunnelOpen()

	tunneledDatabase, err := databases.OpenTunnel(tunnelCtx, databases.OpenTunnelSpec{
		Database:  db,
		Logger:    logger,
		Encryptor: b.fieldEncryptor,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to open ssh tunnel to the source cluster", "error", err)

		return nil, false
	}

	return &backupContext{
		Config:    cfg,
		Database:  tunneledDatabase.GetDatabaseThroughTunnel(),
		Storage:   storage,
		MasterKey: masterKey,
		tunnel:    tunneledDatabase,
	}, true
}

func (b *PhysicalBackuper) resolveParentManifest(
	incrBackup *physical_models.PhysicalIncrementalBackup,
) (postgresql_executor.ParentManifestRef, error) {
	if incrBackup.ParentIncrementalBackupID != nil {
		parent, lookupErr := b.incrRepo.FindByID(*incrBackup.ParentIncrementalBackupID)
		if lookupErr != nil {
			return postgresql_executor.ParentManifestRef{}, fmt.Errorf("look up parent incr: %w", lookupErr)
		}

		if parent == nil || parent.ManifestFileName == nil || parent.StopLSN == nil {
			return postgresql_executor.ParentManifestRef{}, errors.New(
				"parent incremental row missing or has no manifest_file_name / stop_lsn",
			)
		}

		return postgresql_executor.ParentManifestRef{
			BackupID:   parent.ID,
			FileName:   *parent.ManifestFileName,
			Encryption: parent.Encryption,
			Salt:       derefString(parent.ManifestEncryptionSalt),
			IV:         derefString(parent.ManifestEncryptionIV),
			StopLSN:    *parent.StopLSN,
		}, nil
	}

	parent, lookupErr := b.fullRepo.FindByID(incrBackup.RootFullBackupID)
	if lookupErr != nil {
		return postgresql_executor.ParentManifestRef{}, fmt.Errorf("look up root full: %w", lookupErr)
	}

	if parent == nil || parent.ManifestFileName == nil || parent.StopLSN == nil {
		return postgresql_executor.ParentManifestRef{}, errors.New(
			"root full row missing or has no manifest_file_name / stop_lsn")
	}

	return postgresql_executor.ParentManifestRef{
		BackupID:   parent.ID,
		FileName:   *parent.ManifestFileName,
		Encryption: parent.Encryption,
		Salt:       derefString(parent.ManifestEncryptionSalt),
		IV:         derefString(parent.ManifestEncryptionIV),
		StopLSN:    *parent.StopLSN,
	}, nil
}

// The size only feeds telemetry, and a cluster whose pg_hba.conf grants "host replication" but not
// "host all" backs up fine while refusing this probe, so it must never fail a backup. It runs
// before the executor and the connection string carries no connect_timeout, so it also gets its
// own deadline — a stalled source must not hold up the backup it is measuring.
func (b *PhysicalBackuper) getSourceClusterSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	sourceDatabase *databases.Database,
) *float64 {
	probeCtx, cancelProbe := context.WithTimeout(ctx, clusterSizeProbeTimeout)
	defer cancelProbe()

	clusterSizeMb, err := sourceDatabase.PostgresqlPhysical.GetClusterSizeMb(
		probeCtx,
		logger,
		b.fieldEncryptor,
	)
	if err != nil {
		logger.WarnContext(ctx, "failed to measure source cluster size", "error", err)

		return nil
	}

	return &clusterSizeMb
}

func (b *PhysicalBackuper) persistFullResult(
	ctx context.Context,
	fullBackup *physical_models.PhysicalFullBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
	rawSizeMb *float64,
) error {
	fullBackup.Status = backupResult.Status
	fullBackup.ErrorReason = backupResult.ErrorReason
	fullBackup.FailMessage = nilOrPtr(backupResult.ErrorMessage)

	if backupResult.Status == physical_enums.PhysicalBackupStatusCompleted {
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
		completedAt = time.Now().UTC()
	}
	fullBackup.CompletedAt = &completedAt

	return b.saveTerminalResultIfInProgress(ctx, terminalPersistSpec{
		DatabaseID: fullBackup.DatabaseID,
		BackupID:   fullBackup.ID,
		StorageID:  fullBackup.StorageID,
		Result:     backupResult,
		LoadStatus: func(tx *gorm.DB) (physical_enums.PhysicalBackupStatus, error) {
			var current physical_models.PhysicalFullBackup

			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Select("status").
				Where("id = ?", fullBackup.ID).
				First(&current).Error; err != nil {
				return "", err
			}

			return current.Status, nil
		},
		Save: func(tx *gorm.DB) error { return tx.Save(fullBackup).Error },
	})
}

func (b *PhysicalBackuper) persistIncrResult(
	ctx context.Context,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
) error {
	incrBackup.Status = backupResult.Status
	incrBackup.ErrorReason = backupResult.ErrorReason
	incrBackup.FailMessage = nilOrPtr(backupResult.ErrorMessage)

	if backupResult.Status == physical_enums.PhysicalBackupStatusCompleted {
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
		completedAt = time.Now().UTC()
	}
	incrBackup.CompletedAt = &completedAt

	return b.saveTerminalResultIfInProgress(ctx, terminalPersistSpec{
		DatabaseID: incrBackup.DatabaseID,
		BackupID:   incrBackup.ID,
		StorageID:  incrBackup.StorageID,
		Result:     backupResult,
		LoadStatus: func(tx *gorm.DB) (physical_enums.PhysicalBackupStatus, error) {
			var current physical_models.PhysicalIncrementalBackup

			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Select("status").
				Where("id = ?", incrBackup.ID).
				First(&current).Error; err != nil {
				return "", err
			}

			return current.Status, nil
		},
		Save: func(tx *gorm.DB) error { return tx.Save(incrBackup).Error },
	})
}

// The current attempt's row is already IN_PROGRESS in the table (claimAndInsert
// writes it before the backuper runs), so it is dropped before counting.
// Requiring the exact remaining count matters: on a young chain a shorter slice
// would make "every one of them failed" vacuously true and close the chain an
// attempt early. A read failure answers false, because losing a chain to a
// transient database error is worse than granting one retry too many.
//
// Only the executor-result path asks this. The pre-executor finalize paths stay
// plain ERROR deliberately: they return before any notification is sent, and a
// chain that closes without telling anyone is worse than one that closes an
// attempt late.
func (b *PhysicalBackuper) isIncrRetryBudgetExhausted(
	ctx context.Context,
	logger *slog.Logger,
	incrBackup *physical_models.PhysicalIncrementalBackup,
) bool {
	recentBackups, err := b.incrRepo.FindNewestAnyStatusByRootFull(
		incrBackup.RootFullBackupID, maxConsecutiveFailedIncrTries)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read the chain's recent incrementals", "error", err)

		return false
	}

	priorBackups := slices.DeleteFunc(recentBackups,
		func(backup *physical_models.PhysicalIncrementalBackup) bool {
			return backup.ID == incrBackup.ID
		})

	if len(priorBackups) != maxConsecutiveFailedIncrTries-1 {
		return false
	}

	return !slices.ContainsFunc(priorBackups,
		func(backup *physical_models.PhysicalIncrementalBackup) bool {
			return backup.Status != physical_enums.PhysicalBackupStatusError
		})
}

// saveTerminalResultIfInProgress writes the mutated backup row and releases its
// in-flight claim in one transaction, but only while the row is still
// IN_PROGRESS. A backuper can return after a restart-recovery sweep already
// moved the row to a terminal status — and possibly handed the database to a
// fresh backup. Persisting then would resurrect a superseded backup, so the
// guarded read (locked FOR UPDATE to serialize against the sweep's conditional
// update) skips the write instead. The claim delete is
// scoped to backupID so it can never remove the newer backup's claim.
type terminalPersistSpec struct {
	DatabaseID uuid.UUID
	BackupID   uuid.UUID
	StorageID  uuid.UUID
	Result     postgresql_executor.PhysicalBackupResult
	LoadStatus func(tx *gorm.DB) (physical_enums.PhysicalBackupStatus, error)
	Save       func(tx *gorm.DB) error
}

// The files an attempt wrote are kept only by the transaction that publishes it.
// A failed attempt, and a superseded one whose row a later attempt already took,
// both hand their files back to cleanup in that same transaction.
func (b *PhysicalBackuper) saveTerminalResultIfInProgress(ctx context.Context, spec terminalPersistSpec) error {
	superseded := false

	err := storage.GetDb().Transaction(func(tx *gorm.DB) error {
		status, err := spec.LoadStatus(tx)
		if err != nil {
			return err
		}

		if status != physical_enums.PhysicalBackupStatusInProgress {
			superseded = true

			return b.fileStore.RequestFileDeletions(ctx, tx, b.attemptFileReferences(spec))
		}

		if spec.Result.Status == physical_enums.PhysicalBackupStatusCompleted {
			if err := b.fileStore.ConfirmFileWrites(ctx, tx, spec.Result.Receipts); err != nil {
				return err
			}
		} else if err := b.fileStore.RequestFileDeletions(ctx, tx, b.attemptFileReferences(spec)); err != nil {
			return err
		}

		if err := spec.Save(tx); err != nil {
			return err
		}

		return tx.Delete(
			&physical_models.PhysicalInFlightBackup{},
			"database_id = ? AND backup_id = ?",
			spec.DatabaseID,
			spec.BackupID,
		).Error
	})
	if err != nil {
		return err
	}

	if superseded {
		b.logger.Warn("backup row no longer in progress; skipping terminal persist", "backup_id", spec.BackupID)
	}

	return nil
}

// The attempt owns its artifact plus the two sidecars derived from it. Requesting
// a name nothing wrote costs one idempotent provider call and keeps the caller
// from having to know which stage the attempt reached.
func (b *PhysicalBackuper) attemptFileReferences(
	spec terminalPersistSpec,
) []storage_files.StoredFileReference {
	if spec.Result.FileName == "" {
		return nil
	}

	references := []storage_files.StoredFileReference{
		{StorageID: spec.StorageID, FileName: spec.Result.FileName},
		{StorageID: spec.StorageID, FileName: spec.Result.FileName + metadataSuffix},
	}

	if spec.Result.ManifestFileName != "" {
		references = append(references,
			storage_files.StoredFileReference{StorageID: spec.StorageID, FileName: spec.Result.ManifestFileName})
	}

	return references
}

func (b *PhysicalBackuper) finalizeFullAsError(
	fullBackup *physical_models.PhysicalFullBackup,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	r := reason

	fullBackup.Status = physical_enums.PhysicalBackupStatusError
	fullBackup.ErrorReason = &r
	fullBackup.FailMessage = nilOrPtr(message)

	logger := b.logger.With("backup_id", fullBackup.ID, "database_id", fullBackup.DatabaseID)

	if err := b.fullRepo.Save(fullBackup); err != nil {
		logger.Error("failed to flip full row to ERROR", "error", err)
	}

	logger.Error(fmt.Sprintf("physical full backup failed: %s, %s", reason, message))

	// A claim that is not released leaves the database's single in-flight slot occupied, so the
	// scheduler will never start another backup for it.
	if err := b.inFlightRepo.ReleaseOwned(fullBackup.DatabaseID, fullBackup.ID); err != nil {
		logger.Error("failed to release the in-flight claim", "error", err)
	}
}

func (b *PhysicalBackuper) finalizeIncrAsError(
	incrBackup *physical_models.PhysicalIncrementalBackup,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	b.finalizeIncrWithStatus(incrBackup, physical_enums.PhysicalBackupStatusError, reason, message)
}

// finalizeIncrAsChainBroken closes the chain instead of marking a transient
// failure. Use it for irreversible conditions (a missing parent manifest,
// expired summaries) where retrying the same INCR is futile: CHAIN_BROKEN makes
// the scheduler re-anchor on a fresh FULL, whereas ERROR would keep the chain
// extendable and retry the doomed INCR forever. A run of maxConsecutiveFailedIncrTries
// ordinary failures reaches the same verdict through isIncrRetryBudgetExhausted.
func (b *PhysicalBackuper) finalizeIncrAsChainBroken(
	incrBackup *physical_models.PhysicalIncrementalBackup,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	b.finalizeIncrWithStatus(incrBackup, physical_enums.PhysicalBackupStatusChainBroken, reason, message)
}

func (b *PhysicalBackuper) finalizeIncrWithStatus(
	incrBackup *physical_models.PhysicalIncrementalBackup,
	status physical_enums.PhysicalBackupStatus,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	r := reason

	incrBackup.Status = status
	incrBackup.ErrorReason = &r
	incrBackup.FailMessage = nilOrPtr(message)

	logger := b.logger.With("backup_id", incrBackup.ID, "database_id", incrBackup.DatabaseID)

	if err := b.incrRepo.Save(incrBackup); err != nil {
		logger.Error("failed to flip incr row to terminal status", "status", status, "error", err)
	}

	// CHAIN_BROKEN is not a retry: it forces the next tick to open a fresh FULL, so it changes what
	// the scheduler does next and an operator needs to see why.
	if status == physical_enums.PhysicalBackupStatusChainBroken {
		logger.Warn(fmt.Sprintf("physical backup chain broken: %s, %s", reason, message))
	} else {
		logger.Error(fmt.Sprintf("physical incremental backup failed: %s, %s", reason, message))
	}

	if err := b.inFlightRepo.ReleaseOwned(incrBackup.DatabaseID, incrBackup.ID); err != nil {
		logger.Error("failed to release the in-flight claim", "error", err)
	}
}

func (b *PhysicalBackuper) sendFullBackupNotification(
	ctx context.Context,
	cfg *backups_config_physical.PhysicalBackupConfig,
	db *databases.Database,
	fullBackup *physical_models.PhysicalFullBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
) {
	notificationType, title, message := classifyFullBackupNotification(db, fullBackup, backupResult, b.workspaceService)
	if notificationType == "" {
		return
	}

	if !slices.Contains(cfg.SendNotificationsOn, notificationType) {
		return
	}

	notification := notifier_models.Notification{
		Type:    toNotificationType(notificationType),
		Heading: title,
		Message: message,
	}

	for _, notifier := range db.Notifiers {
		b.notificationSender.SendNotification(ctx, &notifier, notification)
	}
}

func (b *PhysicalBackuper) sendIncrBackupNotification(
	ctx context.Context,
	cfg *backups_config_physical.PhysicalBackupConfig,
	db *databases.Database,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
) {
	notificationType, title, message := classifyIncrBackupNotification(db, incrBackup, backupResult, b.workspaceService)
	if notificationType == "" {
		return
	}

	if !slices.Contains(cfg.SendNotificationsOn, notificationType) {
		return
	}

	notification := notifier_models.Notification{
		Type:    toNotificationType(notificationType),
		Heading: title,
		Message: message,
	}

	for _, notifier := range db.Notifiers {
		b.notificationSender.SendNotification(ctx, &notifier, notification)
	}
}

func toNotificationType(
	backupNotificationType backups_config_physical.BackupNotificationType,
) notifier_models.NotificationType {
	if backupNotificationType == backups_config_physical.NotificationBackupSuccess {
		return notifier_models.NotificationTypeBackupSuccess
	}

	return notifier_models.NotificationTypeBackupFailed
}

func classifyFullBackupNotification(
	db *databases.Database,
	fullBackup *physical_models.PhysicalFullBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
	workspaceService *workspaces_services.WorkspaceService,
) (backups_config_physical.BackupNotificationType, string, string) {
	workspaceName := "unknown"
	if db.WorkspaceID != nil {
		if ws, err := workspaceService.GetWorkspaceByID(*db.WorkspaceID); err == nil {
			workspaceName = ws.Name
		}
	}

	switch fullBackup.Status {
	case physical_enums.PhysicalBackupStatusCompleted:
		return backups_config_physical.NotificationBackupSuccess,
			fmt.Sprintf("Physical FULL completed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				fullBackup.ID, backupResult.BackupSizeMb, backupResult.BackupDurationMs)

	case physical_enums.PhysicalBackupStatusError:
		return backups_config_physical.NotificationBackupFailed,
			fmt.Sprintf("Physical FULL failed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				fullBackup.ID, reasonOrEmpty(fullBackup.ErrorReason), backupResult.ErrorMessage)

	case physical_enums.PhysicalBackupStatusChainBroken:
		return backups_config_physical.NotificationChainBroken,
			fmt.Sprintf("Physical FULL chain-broken for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				fullBackup.ID, reasonOrEmpty(fullBackup.ErrorReason), backupResult.ErrorMessage)
	}

	return "", "", ""
}

func classifyIncrBackupNotification(
	db *databases.Database,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
	workspaceService *workspaces_services.WorkspaceService,
) (backups_config_physical.BackupNotificationType, string, string) {
	workspaceName := "unknown"
	if db.WorkspaceID != nil {
		if ws, err := workspaceService.GetWorkspaceByID(*db.WorkspaceID); err == nil {
			workspaceName = ws.Name
		}
	}

	switch incrBackup.Status {
	case physical_enums.PhysicalBackupStatusCompleted:
		return backups_config_physical.NotificationBackupSuccess,
			fmt.Sprintf("Physical INCR completed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				incrBackup.ID, backupResult.BackupSizeMb, backupResult.BackupDurationMs)

	case physical_enums.PhysicalBackupStatusError:
		return backups_config_physical.NotificationBackupFailed,
			fmt.Sprintf("Physical INCR failed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				incrBackup.ID, reasonOrEmpty(incrBackup.ErrorReason), backupResult.ErrorMessage)

	case physical_enums.PhysicalBackupStatusChainBroken:
		return backups_config_physical.NotificationChainBroken,
			fmt.Sprintf("Physical INCR chain-broken for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				incrBackup.ID, reasonOrEmpty(incrBackup.ErrorReason), backupResult.ErrorMessage)
	}

	return "", "", ""
}

func reasonOrEmpty(r *physical_enums.PhysicalBackupErrorReason) string {
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
