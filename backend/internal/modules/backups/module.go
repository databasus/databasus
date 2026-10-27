// Package backups owns how databases are backed up: the backup config of each database, the logical
// backups of a database, and the move of a database to another workspace with its storage and
// notifiers. Other modules reach it only through this file.
package backups

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/auditing"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/download"
	backupshttpapi "databasus-backend/internal/modules/backups/internal/httpapi"
	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/backups/internal/module"
	"databasus-backend/internal/modules/backups/internal/pipeline"
	"databasus-backend/internal/modules/backups/internal/restore"
	"databasus-backend/internal/modules/backups/internal/transfer"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type (
	// Method is how a database is backed up: physically for a physical PostgreSQL database, logically
	// for every other one.
	Method = config.Method
	// BackupConfig holds what the configs of both methods share.
	BackupConfig = config.BackupConfig
	// AnyBackupConfig is a LogicalBackupConfig or a PhysicalBackupConfig.
	AnyBackupConfig      = config.AnyBackupConfig
	LogicalBackupConfig  = config.LogicalBackupConfig
	PhysicalBackupConfig = config.PhysicalBackupConfig

	Encryption                 = config.Encryption
	NotificationTrigger        = config.NotificationTrigger
	GfsRetention               = config.GfsRetention
	LogicalRetention           = config.LogicalRetention
	LogicalRetentionPolicy     = config.LogicalRetentionPolicy
	RetentionPeriod            = config.RetentionPeriod
	PhysicalRetention          = config.PhysicalRetention
	PhysicalRetentionPolicy    = config.PhysicalRetentionPolicy
	FullBackupsRetention       = config.FullBackupsRetention
	FullBackupsRetentionPolicy = config.FullBackupsRetentionPolicy

	// PhysicalBackupConfigSaved is published after the transaction that saves a physical config commits,
	// and its handler's error becomes the save's error. It lets the old physical backups react to a saved
	// config until they move into this module.
	PhysicalBackupConfigSaved = config.PhysicalBackupConfigSaved

	PhysicalBackupConfigSavedHandler = config.PhysicalBackupConfigSavedHandler

	// UserStream is the one heavy stream a user may run at a time: a backup file download or the
	// restore set of a physical backup. Its holder refreshes it every UserStreamHeartbeatInterval and
	// releases it when the stream ends.
	UserStream = download.UserStream

	LogicalBackup                   = logical.LogicalBackup
	LogicalBackupStatus             = logical.Status
	LogicalBackupVerificationStatus = logical.VerificationStatus
	// LogicalBackupCompleted is published after the publication of a logical backup commits.
	LogicalBackupCompleted        = logical.BackupCompleted
	LogicalBackupCompletedHandler = logical.BackupCompletedHandler
)

const (
	LogicalBackupStatusInProgress = logical.StatusInProgress
	LogicalBackupStatusCompleted  = logical.StatusCompleted
	LogicalBackupStatusFailed     = logical.StatusFailed
	LogicalBackupStatusCanceled   = logical.StatusCanceled

	LogicalBackupVerificationStatusNotVerified        = logical.VerificationStatusNotVerified
	LogicalBackupVerificationStatusVerifiedSuccessful = logical.VerificationStatusVerifiedSuccessful
	LogicalBackupVerificationStatusVerificationFailed = logical.VerificationStatusVerificationFailed
)

const UserStreamHeartbeatInterval = download.UserStreamHeartbeatInterval

// ErrUserStreamHeld answers 409 already_running while another stream of the user runs.
var ErrUserStreamHeld = download.ErrUserStreamHeld

const (
	MethodLogical  = config.MethodLogical
	MethodPhysical = config.MethodPhysical

	EncryptionNone      = config.EncryptionNone
	EncryptionEncrypted = config.EncryptionEncrypted

	NotificationTriggerBackupFailed  = config.NotificationTriggerBackupFailed
	NotificationTriggerBackupSuccess = config.NotificationTriggerBackupSuccess
	NotificationTriggerChainBroken   = config.NotificationTriggerChainBroken
	NotificationTriggerWalGap        = config.NotificationTriggerWalGap

	LogicalRetentionPolicyTimePeriod = config.LogicalRetentionPolicyTimePeriod
	LogicalRetentionPolicyCount      = config.LogicalRetentionPolicyCount
	LogicalRetentionPolicyGfs        = config.LogicalRetentionPolicyGfs

	RetentionPeriodDay     = config.RetentionPeriodDay
	RetentionPeriodWeek    = config.RetentionPeriodWeek
	RetentionPeriodMonth   = config.RetentionPeriodMonth
	RetentionPeriod3Months = config.RetentionPeriod3Months
	RetentionPeriod6Months = config.RetentionPeriod6Months
	RetentionPeriodYear    = config.RetentionPeriodYear
	RetentionPeriod2Years  = config.RetentionPeriod2Years
	RetentionPeriod3Years  = config.RetentionPeriod3Years
	RetentionPeriod4Years  = config.RetentionPeriod4Years
	RetentionPeriod5Years  = config.RetentionPeriod5Years
	RetentionPeriodForever = config.RetentionPeriodForever

	PhysicalRetentionPolicyChains               = config.PhysicalRetentionPolicyChains
	PhysicalRetentionPolicyFullBackups          = config.PhysicalRetentionPolicyFullBackups
	PhysicalRetentionPolicyChainsAndFullBackups = config.PhysicalRetentionPolicyChainsAndFullBackups

	FullBackupsRetentionPolicyLastN = config.FullBackupsRetentionPolicyLastN
	FullBackupsRetentionPolicyGfs   = config.FullBackupsRetentionPolicyGfs
)

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)
	// Start fails the backups the previous run left in progress and keeps ctx, the application's
	// context, from which every backup run derives its own. internal/app calls it before the jobs
	// start, and an error stops the application.
	Start(ctx context.Context) error
	GetJobs() []background.Job
	// WaitForRuns returns once every backup run has written its outcome, or with ctx's error.
	WaitForRuns(ctx context.Context) error

	// The getters return the saved config or the default of the database's method, and write nothing.
	// A database of the other method is an error.
	GetLogicalBackupConfig(ctx context.Context, databaseID uuid.UUID) (LogicalBackupConfig, error)
	// LockAndGetLogicalBackupConfig also holds a share lock on the saved config until the caller's
	// transaction ends. A backup start takes it before it inserts its row, so a save that changes the
	// storage and removes the database's backups never misses a backup that starts meanwhile.
	LockAndGetLogicalBackupConfig(ctx context.Context, databaseID uuid.UUID) (LogicalBackupConfig, error)
	GetPhysicalBackupConfig(ctx context.Context, databaseID uuid.UUID) (PhysicalBackupConfig, error)
	// The lists hold only saved configs, so a database whose config was never saved runs no backup.
	ListEnabledLogicalBackupConfigs(ctx context.Context) ([]LogicalBackupConfig, error)
	ListEnabledPhysicalBackupConfigs(ctx context.Context) ([]PhysicalBackupConfig, error)
	// SaveBackupConfig is the save of the contract's updateBackupConfig. It serves the old test helpers
	// and the seed of the dev database.
	SaveBackupConfig(ctx context.Context, actor users.Actor, config AnyBackupConfig) (AnyBackupConfig, error)

	// The force requests each change one column of a saved physical config. A clear given the request
	// it served leaves a newer request in place.
	RequestFullBackup(ctx context.Context, databaseID uuid.UUID) error
	ClearFullBackupRequest(ctx context.Context, databaseID uuid.UUID, requestedAt *time.Time) error
	RequestIncrementalBackup(ctx context.Context, databaseID uuid.UUID) error
	ClearIncrementalBackupRequest(ctx context.Context, databaseID uuid.UUID, requestedAt *time.Time) error

	OnStorageTransferring(ctx context.Context, event storages.StorageTransferring) error
	OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error
	OnDatabaseBackupTypeChanged(ctx context.Context, event databases.DatabaseBackupTypeChanged) error
	OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error

	// The logical backup reads and the verification status serve the old verification runs and
	// telemetry until they move into modules. They check no access. GetLogicalBackups leaves out the
	// IDs that match no backup.
	GetLogicalBackups(ctx context.Context, backupIDs []uuid.UUID) ([]LogicalBackup, error)
	GetLatestCompletedLogicalBackup(ctx context.Context, databaseID uuid.UUID) (LogicalBackup, bool, error)
	HasSuccessfulLogicalBackupSince(ctx context.Context, databaseID uuid.UUID, since time.Time) (bool, error)
	// OpenLogicalBackupFile opens the decrypted file of the backup.
	OpenLogicalBackupFile(ctx context.Context, backupID uuid.UUID) (io.ReadCloser, error)
	SetLogicalBackupVerificationStatus(
		ctx context.Context,
		backupID uuid.UUID,
		status LogicalBackupVerificationStatus,
	) error
	// IsLogicalBackupSchedulerRunning serves the old system health check: false when the scheduler
	// made no pass in the last five minutes.
	IsLogicalBackupSchedulerRunning(ctx context.Context) bool

	// The user streams serve the old physical restore stream until it moves into this module.
	AcquireUserStream(ctx context.Context, userID uuid.UUID) (UserStream, error)
	IsUserStreamHeld(ctx context.Context, userID uuid.UUID) (bool, error)

	// The subscription is made by internal/app while it builds the application. The handlers run in
	// the order they subscribed. The event goes away when the physical backups move into this module.
	SubscribeToPhysicalBackupConfigSaved(handler PhysicalBackupConfigSavedHandler)
	// The handlers of LogicalBackupCompleted run after the publication's commit, in the run of the
	// backup; an error is logged.
	SubscribeToLogicalBackupCompleted(handler LogicalBackupCompletedHandler)
}

type Deps struct {
	DB         *gorm.DB
	Transactor *database.Transactor
	Logger     *slog.Logger
	Clock      clock.Source
	AuditLog   *audit.Log
	MasterKey  *crypto.MasterKey
	// ClientToolsDir holds the client tool bundles of every engine.
	ClientToolsDir string
	// TempDir holds the archive of a PostgreSQL restore that needs it as a file.
	TempDir string
	// Cancellations is the process-wide registry the old physical code shares until it moves into
	// this module.
	Cancellations *background.CancellationRegistry
	// Cache holds the user streams, so they expire on their own when a stream dies.
	Cache      cache.Store
	Users      users.Module
	Workspaces workspaces.Module
	Notifiers  notifiers.Module
	Storages   storages.Module
	Databases  databases.Module
}

func New(deps Deps) Module {
	var logicalService *logical.Service

	configService := config.NewService(config.Deps{
		Repositories: config.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Logger:       deps.Logger,
		Clock:        deps.Clock,
		Workspaces:   deps.Workspaces,
		Storages:     deps.Storages,
		Databases:    deps.Databases,
		LogicalBackups: config.LogicalBackupRemoverFunc(func(ctx context.Context, databaseID uuid.UUID) error {
			return logicalService.RemoveBackupsForStorageChange(ctx, databaseID)
		}),
	})

	// One group runs the backups, the restores and the download heartbeats, so the shutdown waits for
	// every terminal write and every released user stream.
	runs := background.New(background.Deps{Logger: deps.Logger})

	backupPipeline := pipeline.New(pipeline.Deps{
		Databases:      deps.Databases,
		Storages:       deps.Storages,
		MasterKey:      deps.MasterKey,
		ClientToolsDir: deps.ClientToolsDir,
		TempDir:        deps.TempDir,
	})
	restoreRepositories := restore.NewRepositories(deps.DB)
	recorder := auditing.NewRecorder(deps.AuditLog, deps.Users)

	logicalService = logical.NewService(logical.Deps{
		Repositories:  logical.NewRepositories(deps.DB),
		Transactor:    deps.Transactor,
		Logger:        deps.Logger,
		Clock:         deps.Clock,
		Configs:       configService,
		Pipeline:      backupPipeline,
		Runs:          runs,
		Cancellations: deps.Cancellations,
		Audit:         recorder,
		Workspaces:    deps.Workspaces,
		Databases:     deps.Databases,
		Storages:      deps.Storages,
		Notifiers:     deps.Notifiers,
		Restores:      restoreRepositories,
	})

	userStreams := download.NewUserStreams(deps.Cache)

	downloadService := download.NewService(download.Deps{
		Repositories:   download.NewRepositories(deps.DB),
		Transactor:     deps.Transactor,
		Logger:         deps.Logger,
		Clock:          deps.Clock,
		LogicalBackups: logicalService,
		Streams:        userStreams,
		Heartbeats:     runs,
		Audit:          recorder,
	})

	restoreService := restore.NewService(restore.Deps{
		Repositories:   restoreRepositories,
		Transactor:     deps.Transactor,
		Logger:         deps.Logger,
		Clock:          deps.Clock,
		LogicalBackups: logicalService,
		Pipeline:       backupPipeline,
		Runs:           runs,
		Cancellations:  deps.Cancellations,
		Audit:          recorder,
		Workspaces:     deps.Workspaces,
		TempDir:        deps.TempDir,
	})

	return module.New(module.Deps{
		Configs:        configService,
		LogicalBackups: logicalService,
		Downloads:      downloadService,
		Restores:       restoreService,
		UserStreams:    userStreams,
		Runs:           runs,
		Handler: backupshttpapi.NewHandler(backupshttpapi.Deps{
			Configs: configService,
			Transfers: transfer.NewService(transfer.Deps{
				Configs:    configService,
				Transactor: deps.Transactor,
				Workspaces: deps.Workspaces,
				Storages:   deps.Storages,
				Databases:  deps.Databases,
				Notifiers:  deps.Notifiers,
			}),
			LogicalBackups: logicalService,
			Downloads:      downloadService,
			Restores:       restoreService,
		}),
	})
}
