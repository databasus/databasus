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
	"databasus-backend/internal/lib/shutdown"
	"databasus-backend/internal/modules/backups/internal/auditing"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/configsave"
	"databasus-backend/internal/modules/backups/internal/download"
	backupshttpapi "databasus-backend/internal/modules/backups/internal/httpapi"
	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/backups/internal/module"
	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/walstreaming"
	"databasus-backend/internal/modules/backups/internal/pipeline"
	"databasus-backend/internal/modules/backups/internal/restore"
	"databasus-backend/internal/modules/backups/internal/transfer"
	"databasus-backend/internal/modules/backups/internal/usage"
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

	LogicalBackup                   = logical.LogicalBackup
	LogicalBackupVerificationStatus = logical.VerificationStatus
	// LogicalBackupCompleted is published after the publication of a logical backup commits.
	LogicalBackupCompleted        = logical.BackupCompleted
	LogicalBackupCompletedHandler = logical.BackupCompletedHandler
	LogicalBackupsDeleting        = logical.BackupsDeleting
	LogicalBackupsDeletingHandler = logical.BackupsDeletingHandler
)

const (
	LogicalBackupStatusCompleted = logical.StatusCompleted

	LogicalBackupVerificationStatusVerifiedSuccessful = logical.VerificationStatusVerifiedSuccessful
	LogicalBackupVerificationStatusVerificationFailed = logical.VerificationStatusVerificationFailed
)

// ErrUserStreamHeld answers 409 already_running while another stream of the user runs.
var ErrUserStreamHeld = download.ErrUserStreamHeld

const (
	MethodLogical  = config.MethodLogical
	MethodPhysical = config.MethodPhysical
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

	OnStorageTransferring(ctx context.Context, event storages.StorageTransferring) error
	OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error
	OnDatabaseBackupTypeChanged(ctx context.Context, event databases.DatabaseBackupTypeChanged) error
	OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error

	// The logical backup reads and the verification status serve the verification module and
	// telemetry. They check no access. GetLogicalBackups leaves out the
	// IDs that match no backup.
	GetLogicalBackups(ctx context.Context, backupIDs []uuid.UUID) ([]LogicalBackup, error)
	GetLatestCompletedLogicalBackups(ctx context.Context, databaseIDs []uuid.UUID) (map[uuid.UUID]LogicalBackup, error)
	GetLatestCompletedPhysicalFullBackups(
		ctx context.Context,
		databaseIDs []uuid.UUID,
	) (map[uuid.UUID]PhysicalBackup, error)
	GetLatestCompletedLogicalBackup(ctx context.Context, databaseID uuid.UUID) (LogicalBackup, bool, error)
	// OpenLogicalBackupFile opens the decrypted file of the backup.
	OpenLogicalBackupFile(ctx context.Context, backupID uuid.UUID) (io.ReadCloser, error)
	SetLogicalBackupVerificationStatus(
		ctx context.Context,
		backupID uuid.UUID,
		status LogicalBackupVerificationStatus,
	) error

	// The handlers of LogicalBackupCompleted run after the publication's commit, in the run of the
	// backup; an error is logged.
	SubscribeToLogicalBackupCompleted(handler LogicalBackupCompletedHandler)
	// Deleting handlers run in the caller's transaction; an error rolls the deletion back.
	SubscribeToLogicalBackupsDeleting(handler LogicalBackupsDeletingHandler)
}

type Deps struct {
	ShutdownBudget         *shutdown.Budget
	DB                     *gorm.DB
	Transactor             *database.Transactor
	Logger                 *slog.Logger
	Clock                  clock.Source
	PhysicalExecutionClock clock.Source
	AuditLog               *audit.Log
	MasterKey              *crypto.MasterKey
	// ClientToolsDir holds the client tool bundles of every engine.
	ClientToolsDir string
	// TempDir holds the archive of a PostgreSQL restore that needs it as a file.
	TempDir          string
	WalQueueDir      string
	WalStreamTimings WalStreamTimings
	// Cache holds the user streams, so they expire on their own when a stream dies.
	Cache      cache.Store
	Users      users.Module
	Workspaces workspaces.Module
	Notifiers  notifiers.Module
	Storages   storages.Module
	Databases  databases.Module
}

func New(deps Deps) Module {
	cancellations := background.NewCancellationRegistry()
	configRepositories := config.NewRepositories(deps.DB)

	configService := config.NewService(config.Deps{
		Repositories: configRepositories,
		Logger:       deps.Logger,
		Clock:        deps.Clock,
		Workspaces:   deps.Workspaces,
		Storages:     deps.Storages,
		Databases:    deps.Databases,
	})

	// One group runs the backups, the restores and the download heartbeats, so the shutdown waits for
	// every terminal write and every released user stream.
	runs := background.New(background.Deps{Logger: deps.Logger})

	backupPipeline := pipeline.New(pipeline.Deps{
		ShutdownBudget: deps.ShutdownBudget,
		Databases:      deps.Databases,
		Storages:       deps.Storages,
		MasterKey:      deps.MasterKey,
		ClientToolsDir: deps.ClientToolsDir,
		TempDir:        deps.TempDir,
	})
	restoreRepositories := restore.NewRepositories(deps.DB)
	recorder := auditing.NewRecorder(deps.AuditLog, deps.Users)

	usageService := usage.NewService(usage.Deps{
		Repositories: usage.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Storages:     deps.Storages,
		Workspaces:   deps.Workspaces,
	})

	logicalService := logical.NewService(logical.Deps{
		Usage:         usageService,
		Repositories:  logical.NewRepositories(deps.DB),
		Transactor:    deps.Transactor,
		Logger:        deps.Logger,
		Clock:         deps.Clock,
		Configs:       configService,
		Pipeline:      backupPipeline,
		Runs:          runs,
		Cancellations: cancellations,
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
		ShutdownBudget: deps.ShutdownBudget,
		Repositories:   restoreRepositories,
		Transactor:     deps.Transactor,
		Logger:         deps.Logger,
		Clock:          deps.Clock,
		LogicalBackups: logicalService,
		Pipeline:       backupPipeline,
		Runs:           runs,
		Cancellations:  cancellations,
		Audit:          recorder,
		Workspaces:     deps.Workspaces,
		TempDir:        deps.TempDir,
	})

	physicalService := physical.New(physical.Deps{
		Usage:          usageService,
		ExecutionClock: deps.PhysicalExecutionClock,
		Repository:     catalog.NewRepository(deps.DB),
		Backups:        catalog.NewBackupRepository(deps.DB),
		WalSegments:    catalog.NewWalSegmentRepository(deps.DB),
		WalHistory:     catalog.NewWalHistoryRepository(deps.DB),
		WalStreamers: catalog.NewWalStreamerRepository(
			deps.DB,
		),
		WalStreamSamples: catalog.NewWalStreamSampleRepository(deps.DB),
		Configs:          configService,
		Cancellations:    cancellations,
		Runs:             runs,
		UserStreams:      userStreams,
		Cache:            deps.Cache,
		Databases:        deps.Databases,
		Storages:         deps.Storages,
		Workspaces:       deps.Workspaces,
		Notifiers:        deps.Notifiers,
		Auditing:         recorder,
		Transactor:       deps.Transactor,
		Clock:            deps.Clock,
		MasterKey:        deps.MasterKey,
		Logger:           deps.Logger,
		ClientToolsDir:   deps.ClientToolsDir,
		WalQueueDir:      deps.WalQueueDir,
		WalStreamTimings: deps.WalStreamTimings,
	})

	configSaveService := configsave.NewService(configsave.Deps{
		Repositories:    configRepositories,
		Configs:         configService,
		LogicalBackups:  logicalService,
		PhysicalBackups: physicalService,
		Transactor:      deps.Transactor,
		Logger:          deps.Logger,
		Clock:           deps.Clock,
		Workspaces:      deps.Workspaces,
		Storages:        deps.Storages,
		Databases:       deps.Databases,
	})

	return module.New(module.Deps{
		Physical:       physicalService,
		Configs:        configService,
		LogicalBackups: logicalService,
		Downloads:      downloadService,
		Restores:       restoreService,
		UserStreams:    userStreams,
		Runs:           runs,
		Handler: backupshttpapi.NewHandler(backupshttpapi.Deps{
			Usage:           usageService,
			ConfigSaves:     configSaveService,
			PhysicalBackups: physicalService,
			Configs:         configService,
			Transfers: transfer.NewService(transfer.Deps{
				ConfigSaves: configSaveService,
				Configs:     configService,
				Transactor:  deps.Transactor,
				Workspaces:  deps.Workspaces,
				Storages:    deps.Storages,
				Databases:   deps.Databases,
				Notifiers:   deps.Notifiers,
			}),
			LogicalBackups: logicalService,
			Downloads:      downloadService,
			Restores:       restoreService,
		}),
	})
}

type WalStreamTimings = walstreaming.Timings

func ProductionWalStreamTimings() WalStreamTimings {
	return WalStreamTimings{
		ForcedRotationInterval:    5 * time.Minute,
		ArchiveStalenessThreshold: 30 * time.Minute,
		LocalQueuePauseBytes:      0,
	}
}

type PhysicalBackup = catalog.Backup
