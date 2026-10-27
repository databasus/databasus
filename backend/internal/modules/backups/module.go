// Package backups owns how databases are backed up: the backup config of each database and the move of
// a database to another workspace with its storage and notifiers. Other modules reach it only through
// this file.
package backups

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	backupshttpapi "databasus-backend/internal/modules/backups/internal/httpapi"
	"databasus-backend/internal/modules/backups/internal/module"
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

	// LogicalBackupConfigStorageChanging is published inside the transaction that saves a logical config
	// with another storage. It lets the old logical backups leave together with the save until they move
	// into this module.
	LogicalBackupConfigStorageChanging = config.LogicalBackupConfigStorageChanging
	// PhysicalBackupConfigSaved is published after the transaction that saves a physical config commits,
	// and its handler's error becomes the save's error. It lets the old physical backups react to a saved
	// config until they move into this module.
	PhysicalBackupConfigSaved = config.PhysicalBackupConfigSaved

	LogicalBackupConfigStorageChangingHandler = config.LogicalBackupConfigStorageChangingHandler
	PhysicalBackupConfigSavedHandler          = config.PhysicalBackupConfigSavedHandler
)

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

	// The subscriptions are made by internal/app while it builds the application. The handlers run in
	// the order they subscribed. Both events go away when the logical and the physical backups move into
	// this module.
	SubscribeToLogicalBackupConfigStorageChanging(handler LogicalBackupConfigStorageChangingHandler)
	SubscribeToPhysicalBackupConfigSaved(handler PhysicalBackupConfigSavedHandler)
}

type Deps struct {
	DB         *gorm.DB
	Transactor *database.Transactor
	Logger     *slog.Logger
	Clock      clock.Source
	Workspaces workspaces.Module
	Notifiers  notifiers.Module
	Storages   storages.Module
	Databases  databases.Module
}

func New(deps Deps) Module {
	configService := config.NewService(config.Deps{
		Repositories: config.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Logger:       deps.Logger,
		Clock:        deps.Clock,
		Workspaces:   deps.Workspaces,
		Storages:     deps.Storages,
		Databases:    deps.Databases,
	})

	return module.New(module.Deps{
		Configs: configService,
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
		}),
	})
}
