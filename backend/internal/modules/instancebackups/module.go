// Package instancebackups owns Databasus backups of the internal database: their settings and
// destinations, runs, copies and retention, protection health and incident mail, master-key
// export and recovery of the instance from a Databasus backup file. Other modules reach it only
// through this file.
package instancebackups

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"databasus-backend/internal/engines/postgresql/internaldb"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/lib/lifecycle"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/instancebackups/internal/catalog"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/instancebackups/internal/health"
	instancebackupshttpapi "databasus-backend/internal/modules/instancebackups/internal/httpapi"
	"databasus-backend/internal/modules/instancebackups/internal/keyexport"
	"databasus-backend/internal/modules/instancebackups/internal/module"
	"databasus-backend/internal/modules/instancebackups/internal/recovery"
	"databasus-backend/internal/modules/instancebackups/internal/snapshot"
	"databasus-backend/internal/modules/instancebackups/internal/tokens"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

// RecoveredStateNormalizer ends, in a database a recovery staged, what one module had in progress
// on the lost instance, and returns how many unfinished file writes and deletions it discarded.
type RecoveredStateNormalizer = recovery.StateNormalizer

// LifecycleLock is held shared by every server and console process for its lifetime and exclusive
// by recovery publication. Close releases it.
type LifecycleLock = recovery.LifecycleLock

// StartupDeps is what a process needs before it opens the internal database.
type StartupDeps struct {
	DatabaseDsn            string
	RecoveryStateDirectory string
	SecretKeyPath          string
	DataFolder             string
	ClientToolsDir         string
	Release                string
	Bundle                 database.MigrationBundle
	// DumpPath and MasterKeyPath name the files of a configured startup recovery; both are empty
	// without one.
	DumpPath      string
	MasterKeyPath string
	// NewerSchemaGuidance is the next step shown for a database migrated by a newer version.
	NewerSchemaGuidance string
	// Normalizers end the other modules' recovered work before publication.
	Normalizers []RecoveredStateNormalizer
	Clock       clock.Source
	Logger      *slog.Logger
	InspectDisk func(string) (diskusage.Usage, error)
}

// OpenServerLifecycle runs before the server opens the internal database: it finishes a recovery
// publication a previous process left unfinished and runs a configured startup recovery. A refused
// recovery is logged with its reason and next step and returned, and the server does not start.
func OpenServerLifecycle(ctx context.Context, deps StartupDeps) (*LifecycleLock, error) {
	startup, err := newStartup(deps)
	if err != nil {
		return nil, err
	}

	return startup.OpenForServer(ctx)
}

// PublishAcceptedRecovery runs in a server process after its application stopped and closed its
// connections because root admin accepted a recovery. It publishes the recovery, or ends it as
// QUIESCENCE_FAILED when isDrained is false or another process uses the data directory, and leaves
// lock shared for the application built next.
func PublishAcceptedRecovery(ctx context.Context, deps StartupDeps, lock *LifecycleLock, isDrained bool) error {
	startup, err := newStartup(deps)
	if err != nil {
		return err
	}

	return startup.PublishAccepted(ctx, lock, isDrained)
}

// OpenConsoleLifecycle runs before a console command opens the internal database. It never runs a
// recovery and refuses while a publication is unfinished.
func OpenConsoleLifecycle(deps StartupDeps) (*LifecycleLock, error) {
	startup, err := newStartup(deps)
	if err != nil {
		return nil, err
	}

	return startup.OpenForConsole()
}

func newStartup(deps StartupDeps) (*recovery.Startup, error) {
	server, err := internaldb.ParseConnection(deps.DatabaseDsn)
	if err != nil {
		return nil, err
	}

	return recovery.NewStartup(recovery.StartupDeps{
		Directory:           recovery.NewDirectory(deps.RecoveryStateDirectory),
		Server:              server,
		SecretKeyPath:       deps.SecretKeyPath,
		DataFolder:          deps.DataFolder,
		ClientTools:         pgshared.ClientTools{Root: deps.ClientToolsDir},
		Release:             deps.Release,
		Bundle:              deps.Bundle,
		DumpPath:            deps.DumpPath,
		MasterKeyPath:       deps.MasterKeyPath,
		NewerSchemaGuidance: deps.NewerSchemaGuidance,
		Normalizers:         deps.Normalizers,
		Clock:               deps.Clock,
		Logger:              deps.Logger,
		InspectDisk:         deps.InspectDisk,
	}), nil
}

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)
	// OnStorageDeleting refuses with databasus_backup_in_use a storage or workspace deletion that
	// would delete a selected Databasus backup destination.
	OnStorageDeleting(ctx context.Context, event storages.StorageDeleting) error
	// Start ends the dumps and uploads a stopped process left unfinished, before the jobs run.
	Start(ctx context.Context) error
	GetJobs() []background.Job
}

type Deps struct {
	DB         *gorm.DB
	Transactor *database.Transactor
	Clock      clock.Source
	AuditLog   *audit.Log
	MasterKey  *crypto.MasterKey
	Mailer     mailer.Sender
	// Lifecycle locates the recovery journal and the internal database, and prepares uploaded
	// recoveries exactly as a startup recovery.
	Lifecycle StartupDeps
	// RebuildRequest asks the server process to publish an accepted recovery in place.
	RebuildRequest *lifecycle.RebuildRequest
	// MaxRecoveryArchiveBytes bounds a Databasus backup file uploaded for recovery.
	MaxRecoveryArchiveBytes int64
	// SpoolFolder holds the encrypted files of Databasus backups until every copy ended; nothing
	// else may live there.
	SpoolFolder string
	Users       users.Module
	Workspaces  workspaces.Module
	Storages    storages.Module
	Databases   databases.Module
}

func New(deps Deps) (Module, error) {
	var (
		catalogService *catalog.Service
		healthService  *health.Service
	)

	configService := configs.NewService(configs.Deps{
		Repositories:             configs.NewRepositories(deps.DB),
		Transactor:               deps.Transactor,
		Clock:                    deps.Clock,
		AuditLog:                 deps.AuditLog,
		Users:                    deps.Users,
		Storages:                 deps.Storages,
		Workspaces:               deps.Workspaces,
		Databases:                deps.Databases,
		OnSettingsCommitted:      func(configs.SettingsChange) { catalogService.WakeProduction() },
		OnLastDestinationRemoved: func(ctx context.Context) error { return healthService.OpenStoppedIncident(ctx) },
	})

	tokenService := tokens.NewService(tokens.Deps{Repositories: tokens.NewRepositories(deps.DB), Clock: deps.Clock})

	keyExportService := keyexport.NewService(keyexport.Deps{
		Transactor: deps.Transactor,
		AuditLog:   deps.AuditLog,
		MasterKey:  deps.MasterKey,
		Users:      deps.Users,
		Configs:    configService,
		Tokens:     tokenService,
	})

	recoveryDirectory := recovery.NewDirectory(deps.Lifecycle.RecoveryStateDirectory)
	recoveryJournal := recovery.NewJournalStore(recoveryDirectory)

	recoveryService := recovery.NewService(recovery.ServiceDeps{
		Journal:        recoveryJournal,
		Clock:          deps.Clock,
		Transactor:     deps.Transactor,
		AuditLog:       deps.AuditLog,
		Users:          deps.Users,
		RunningRelease: deps.Lifecycle.Release,
		RebuildRequest: deps.RebuildRequest,
	})

	uploadService, err := newUploadService(deps, recoveryDirectory, recoveryJournal)
	if err != nil {
		return nil, err
	}

	catalogService, err = newCatalogService(deps, configService, tokenService)
	if err != nil {
		return nil, err
	}

	healthService = health.NewService(health.Deps{
		Repositories: health.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Clock:        deps.Clock,
		Logger:       deps.Lifecycle.Logger,
		Mailer:       deps.Mailer,
		Users:        deps.Users,
		Configs:      configService,
		KeyExport:    keyExportService,
		Recoveries:   recoveryService,
	})

	return module.New(module.Deps{
		Handler: instancebackupshttpapi.NewHandler(instancebackupshttpapi.Deps{
			Configs:         configService,
			KeyExport:       keyExportService,
			Status:          healthService,
			Recoveries:      recoveryService,
			RecoveryUploads: uploadService,
			Backups:         catalogService,
		}),
		Configs: configService,
		Catalog: catalogService,
		Jobs: []background.Job{
			{
				Name:       recovery.PreparationJobName,
				Interval:   recovery.PreparationJobInterval,
				Run:        uploadService.RunPass,
				WakeSignal: uploadService.GetWakeSignal(),
			},
			{
				Name:       catalog.ProductionJobName,
				Interval:   catalog.ProductionJobInterval,
				Run:        catalogService.RunProductionPass,
				WakeSignal: catalogService.GetProductionSignal(),
			},
			{
				Name:       catalog.DeliveryJobName,
				Interval:   catalog.DeliveryJobInterval,
				Run:        catalogService.RunDeliveryPass,
				WakeSignal: catalogService.GetDeliverySignal(),
			},
			{
				Name:       health.IncidentJobName,
				Interval:   health.IncidentJobInterval,
				Run:        healthService.RunIncidentPass,
				WakeSignal: nil,
			},
			{
				Name:       catalog.RetentionJobName,
				Interval:   catalog.RetentionJobInterval,
				Run:        catalogService.RunRetentionPass,
				WakeSignal: nil,
			},
		},
	}), nil
}

func newCatalogService(
	deps Deps,
	configService *configs.Service,
	tokenService *tokens.Service,
) (*catalog.Service, error) {
	connection, err := internaldb.ParseConnection(deps.Lifecycle.DatabaseDsn)
	if err != nil {
		return nil, err
	}

	return catalog.NewService(catalog.Deps{
		Repositories: catalog.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Clock:        deps.Clock,
		Logger:       deps.Lifecycle.Logger,
		AuditLog:     deps.AuditLog,
		Users:        deps.Users,
		Storages:     deps.Storages,
		Configs:      configService,
		Producer: snapshot.NewProducer(snapshot.ProducerDeps{
			Repositories: snapshot.NewRepositories(deps.DB),
			Connection:   connection,
			ClientTools:  pgshared.ClientTools{Root: deps.Lifecycle.ClientToolsDir},
			MasterKey:    deps.MasterKey,
			Release:      deps.Lifecycle.Release,
			Bundle:       deps.Lifecycle.Bundle,
			SpoolDir:     deps.SpoolFolder,
			Clock:        deps.Clock,
			Logger:       deps.Lifecycle.Logger,
			InspectDisk:  deps.Lifecycle.InspectDisk,
		}),
		Tokens:  tokenService,
		Release: deps.Lifecycle.Release,
	}), nil
}

func newUploadService(
	deps Deps,
	directory recovery.Directory,
	journal *recovery.JournalStore,
) (*recovery.UploadService, error) {
	server, err := internaldb.ParseConnection(deps.Lifecycle.DatabaseDsn)
	if err != nil {
		return nil, err
	}

	return recovery.NewUploadService(recovery.UploadServiceDeps{
		Directory: directory,
		Journal:   journal,
		Server:    server,
		Preparer: recovery.NewPreparer(recovery.PreparerDeps{
			Server:      server,
			ClientTools: pgshared.ClientTools{Root: deps.Lifecycle.ClientToolsDir},
			Release:     deps.Lifecycle.Release,
			Bundle:      deps.Lifecycle.Bundle,
			Normalizers: deps.Lifecycle.Normalizers,
			Clock:       deps.Clock,
			Logger:      deps.Lifecycle.Logger,
		}),
		Publisher: recovery.NewPublisher(recovery.PublisherDeps{
			Directory:     directory,
			Journal:       journal,
			Server:        server,
			SecretKeyPath: deps.Lifecycle.SecretKeyPath,
			Clock:         deps.Clock,
			Logger:        deps.Lifecycle.Logger,
		}),
		Users:           deps.Users,
		MaxArchiveBytes: deps.MaxRecoveryArchiveBytes,
		RunningRelease:  deps.Lifecycle.Release,
		Clock:           deps.Clock,
		Logger:          deps.Lifecycle.Logger,
		InspectDisk:     deps.Lifecycle.InspectDisk,
	}), nil
}
