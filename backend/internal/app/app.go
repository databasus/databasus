package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/lib/shutdown"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/healthcheck"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/telemetry"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification"
	"databasus-backend/internal/modules/workspaces"
	"databasus-backend/migrations"
)

const olderVersionRefusal = "Returning to an older Databasus version after an upgrade is not supported. " +
	"Start the image of the newer version instead"

type App struct {
	usersModule        users.Module
	workspacesModule   workspaces.Module
	notifiersModule    notifiers.Module
	storagesModule     storages.Module
	databasesModule    databases.Module
	backupsModule      backups.Module
	verificationModule verification.Module
	healthcheckModule  healthcheck.Module
	environment        *Environment
	db                 *gorm.DB
	transactor         *database.Transactor
	logger             *slog.Logger
	// backgroundGroup exists only while Serve runs, so a console command cannot start a loop.
	backgroundGroup *background.Group
	// runner is started only by Serve; a console command and a test application run its passes
	// through RunJobPass.
	jobHealthPolicies  map[string]JobHealthPolicy
	jobs               []background.Job
	runner             *background.Runner
	inspectDisk        func(string) (diskusage.Usage, error)
	clientToolsChecker *clienttools.Checker
	shutdownBudget     *shutdown.Budget
}

// New prepares everything the server and the console commands share: the schema of the current
// version, the data directories, the master key file and the module graph.
func New(ctx context.Context, environment *Environment) (*App, error) {
	appLogger := environment.Logging.GetLogger()

	db, err := database.Open(ctx, environment.Config.DatabaseDsn)
	if err != nil {
		return nil, err
	}

	appLogger.InfoContext(ctx, "running database migrations")

	appliedMigrations, err := database.Migrate(ctx, db, migrations.Files)
	if errors.Is(err, database.ErrSchemaNewerThanBinary) {
		closeDatabase(db)

		err = fmt.Errorf("%w. %s", err, olderVersionRefusal)
		appLogger.ErrorContext(ctx, "refusing to start on a database migrated by a newer version",
			slog.Any("error", err))

		return nil, err
	}

	if err != nil {
		closeDatabase(db)
		appLogger.ErrorContext(ctx, "failed to run migrations", slog.Any("error", err))

		return nil, err
	}

	appLogger.InfoContext(ctx, "database migrations completed successfully",
		slog.Int("applied_count", len(appliedMigrations)))

	err = ensureDirectories([]string{environment.Config.TempFolder, environment.Config.DataFolder})
	if err != nil {
		closeDatabase(db)

		return nil, fmt.Errorf("ensure data directories: %w", err)
	}

	transactor := database.NewTransactor(db)

	masterKey, err := moveLegacyAndOpenMasterKey(ctx, crypto.Deps{
		Transactor:          transactor,
		LegacyKeyRepository: crypto.NewLegacyMasterKeyRepository(db),
		MasterKeyPath:       environment.Config.SecretKeyPath,
		Logger:              appLogger,
	})
	if err != nil {
		closeDatabase(db)
		appLogger.ErrorContext(ctx, "failed to open the master key", slog.Any("error", err))

		return nil, err
	}

	fieldEncryptor := crypto.NewFieldEncryptor(masterKey)

	cacheStore := cache.NewMemoryStore(cache.Deps{
		PayloadBudgetBytes: cache.DefaultPayloadBudgetBytes,
		Clock:              environment.Clock,
	})
	rateLimiter := ratelimit.NewMemoryCounter(ratelimit.Deps{Store: cacheStore, Clock: environment.Clock})

	auditLog := audit.New(audit.Deps{
		DB:          db,
		Clock:       environment.Clock,
		AuditLogger: environment.Logging.GetAuditLogger(),
	})

	usersModule := users.New(users.Deps{
		DB:          db,
		Transactor:  transactor,
		Clock:       environment.Clock,
		Logger:      appLogger,
		MasterKey:   masterKey,
		Mailer:      environment.Mailer,
		AuditLog:    auditLog,
		RateLimiter: rateLimiter,
		Turnstile: users.TurnstileSettings{
			SecretKey: environment.Config.CloudflareTurnstileSecretKey,
			VerifyURL: environment.ExternalEndpoints.TurnstileVerifyURL,
		},
		GitHub: users.GitHubSettings{
			ClientID:     environment.Config.GitHubClientID,
			ClientSecret: environment.Config.GitHubClientSecret,
			AuthURL:      environment.ExternalEndpoints.GitHubAuthURL,
			TokenURL:     environment.ExternalEndpoints.GitHubTokenURL,
			UserURL:      environment.ExternalEndpoints.GitHubUserURL,
			EmailsURL:    environment.ExternalEndpoints.GitHubEmailsURL,
		},
		Google: users.GoogleSettings{
			ClientID:     environment.Config.GoogleClientID,
			ClientSecret: environment.Config.GoogleClientSecret,
			AuthURL:      environment.ExternalEndpoints.GoogleAuthURL,
			TokenURL:     environment.ExternalEndpoints.GoogleTokenURL,
			UserInfoURL:  environment.ExternalEndpoints.GoogleUserInfoURL,
		},
	})

	workspacesModule := workspaces.New(workspaces.Deps{
		DB:         db,
		Transactor: transactor,
		Clock:      environment.Clock,
		Logger:     appLogger,
		Mailer:     environment.Mailer,
		AuditLog:   auditLog,
		Users:      usersModule,
		PublicURL:  environment.Config.DatabasusURL,
	})

	notifiersModule := notifiers.New(notifiers.Deps{
		DB:              db,
		Transactor:      transactor,
		Logger:          appLogger,
		AuditLog:        auditLog,
		FieldEncryptor:  fieldEncryptor,
		Users:           usersModule,
		Workspaces:      workspacesModule,
		DeliveryTimeout: environment.NotificationDeliveryTimeout,
		Endpoints: notifiers.Endpoints{
			SlackPostMessageURL: environment.ExternalEndpoints.SlackPostMessageURL,
			TelegramBotAPIURL:   environment.ExternalEndpoints.TelegramBotAPIURL,
		},
		PublicURL: environment.Config.DatabasusURL,
	})

	storagesModule := storages.New(storages.Deps{
		DB:             db,
		Transactor:     transactor,
		Logger:         appLogger,
		AuditLog:       auditLog,
		FieldEncryptor: fieldEncryptor,
		Users:          usersModule,
		Workspaces:     workspacesModule,
		Clock:          environment.Clock,
		LocalFolders: storages.LocalFolders{
			TempFolder: environment.Config.TempFolder,
			DataFolder: environment.Config.DataFolder,
		},
		FileTimings: environment.StorageFileTimings,
	})

	databasesModule := databases.New(databases.Deps{
		DB:             db,
		Transactor:     transactor,
		Logger:         appLogger,
		AuditLog:       auditLog,
		FieldEncryptor: fieldEncryptor,
		ClientToolsDir: environment.ClientToolsDir,
		Users:          usersModule,
		Workspaces:     workspacesModule,
		Notifiers:      notifiersModule,
	})

	shutdownBudget := shutdown.New(shutdown.Deps{Clock: clock.NewSystem()})
	backupsModule := backups.New(backups.Deps{
		ShutdownBudget:         shutdownBudget,
		PhysicalExecutionClock: clock.NewSystem(),
		DB:                     db,
		Transactor:             transactor,
		Logger:                 appLogger,
		Clock:                  environment.Clock,
		AuditLog:               auditLog,
		MasterKey:              masterKey,
		ClientToolsDir:         environment.ClientToolsDir,
		TempDir:                environment.Config.TempFolder,
		WalQueueDir:            filepath.Join(environment.Config.DataFolder, "wal-queue"),
		WalStreamTimings:       environment.WalStreamTimings,
		Cache:                  cacheStore,
		Users:                  usersModule,
		Workspaces:             workspacesModule,
		Notifiers:              notifiersModule,
		Storages:               storagesModule,
		Databases:              databasesModule,
	})
	verificationModule := verification.New(
		verification.Deps{
			DB:          db,
			Transactor:  transactor,
			Logger:      appLogger,
			Clock:       environment.Clock,
			AuditLog:    auditLog,
			Users:       usersModule,
			Workspaces:  workspacesModule,
			Databases:   databasesModule,
			Backups:     backupsModule,
			Notifiers:   notifiersModule,
			RateLimiter: rateLimiter,
		},
	)

	healthcheckModule := healthcheck.New(healthcheck.Deps{
		DB:         db,
		Transactor: transactor,
		Logger:     appLogger,
		Clock:      environment.Clock,
		AuditLog:   auditLog,
		Users:      usersModule,
		Workspaces: workspacesModule,
		Databases:  databasesModule,
		Notifiers:  notifiersModule,
	})

	telemetryModule := telemetry.New(telemetry.Deps{
		Users:        usersModule,
		Databases:    databasesModule,
		Storages:     storagesModule,
		Notifiers:    notifiersModule,
		Backups:      backupsModule,
		Verification: verificationModule,
		Clock:        environment.Clock,
		Logger:       appLogger,
		Version:      environment.Config.AppVersion,
		ReceiverURL:  environment.ExternalEndpoints.TelemetryReceiverURL,
		IdentityPath: environment.Config.TelemetryInstancePath,
		IsDisabled:   environment.Config.IsDisableAnonymousTelemetry,
	})

	subscribeToWorkspaceDeleting(workspacesModule, databasesModule, notifiersModule, storagesModule)
	subscribeToNotifierTransferring(notifiersModule, databasesModule)
	subscribeToStorageTransferring(storagesModule, backupsModule)
	subscribeToDatabaseEvents(databaseSubscriptions{
		Databases:    databasesModule,
		Backups:      backupsModule,
		Verification: verificationModule,
		Healthcheck:  healthcheckModule,
	})
	subscribeToLogicalBackupEvents(backupsModule, verificationModule)

	jobs := append([]background.Job{newAuditEventsCleanupJob(auditLog, appLogger)}, usersModule.GetJobs()...)
	jobs = append(jobs, storagesModule.GetJobs()...)
	jobs = append(jobs, backupsModule.GetJobs()...)
	jobs = append(jobs, verificationModule.GetJobs()...)
	jobs = append(jobs, healthcheckModule.GetJobs()...)
	jobs = append(jobs, telemetryModule.GetJobs()...)
	runner := background.NewRunner(background.RunnerDeps{Logger: appLogger, Clock: environment.Clock}, jobs...)

	return &App{
		usersModule:        usersModule,
		workspacesModule:   workspacesModule,
		notifiersModule:    notifiersModule,
		storagesModule:     storagesModule,
		databasesModule:    databasesModule,
		backupsModule:      backupsModule,
		verificationModule: verificationModule,
		healthcheckModule:  healthcheckModule,
		environment:        environment,
		db:                 db,
		transactor:         transactor,
		logger:             appLogger,
		backgroundGroup:    nil,
		jobHealthPolicies:  newJobHealthPolicies(jobs, environment.StorageFileTimings),
		jobs:               jobs, runner: runner,
		inspectDisk:        diskusage.Get,
		clientToolsChecker: environment.getClientToolsChecker(),
		shutdownBudget:     shutdownBudget,
	}, nil
}

// StartModules runs the startup work of the modules on ctx, which ends when the application stops.
// Serve calls it before it accepts requests, and a test application before its first request.
func (a *App) StartModules(ctx context.Context) error {
	if err := a.backupsModule.Start(ctx); err != nil {
		return err
	}
	return nil
}

// WaitForBackupRuns returns once every backup run has written its outcome, or with ctx's error.
func (a *App) WaitForBackupRuns(ctx context.Context) error {
	return a.backupsModule.WaitForRuns(ctx)
}

// RunJobPass runs one pass of the named background job and returns its error.
func (a *App) RunJobPass(ctx context.Context, name string) error {
	return a.runner.RunPass(ctx, name)
}

// The persisted key must reach its file before modules can encrypt new values.
func moveLegacyAndOpenMasterKey(ctx context.Context, legacyKeyDeps crypto.Deps) (*crypto.MasterKey, error) {
	err := crypto.MoveLegacyMasterKeyToFile(ctx, legacyKeyDeps)
	if err != nil {
		return nil, fmt.Errorf("move the legacy master key to its file: %w", err)
	}

	return crypto.OpenOrCreateMasterKey(legacyKeyDeps.MasterKeyPath)
}

// Close releases the connections to the internal database. The server never calls it, because the
// process ends with Serve; a test application closes them before its database is dropped.
func (a *App) Close() {
	closeDatabase(a.db)
}

func closeDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

// SubscribeToWorkspaceDeleting adds a handler after the ones subscribed in wiring.go. A test
// application uses it to stand in for a higher module.
func (a *App) SubscribeToWorkspaceDeleting(handler workspaces.WorkspaceDeletingHandler) {
	a.workspacesModule.SubscribeToWorkspaceDeleting(handler)
}

// GetPhysicalBackupTypes reads the backup types of physical databases from the databases module. A test
// application uses it because only the backups module calls it.
func (a *App) GetPhysicalBackupTypes(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]databases.PhysicalBackupType, error) {
	return a.databasesModule.GetPhysicalBackupTypes(ctx, databaseIDs)
}

func (a *App) Notify(ctx context.Context, notifierIDs []uuid.UUID, notification notifiers.Notification) {
	a.notifiersModule.Notify(ctx, notifierIDs, notification)
}

func (a *App) WaitForNotificationDeliveries(ctx context.Context) error {
	return a.notifiersModule.WaitForDeliveries(ctx)
}

func (a *App) WriteStorageFile(
	ctx context.Context,
	reference storages.StoredFileReference,
	file io.Reader,
) (storages.WriteReceipt, error) {
	return a.storagesModule.WriteFile(ctx, reference, file)
}

func (a *App) PublishStorageFile(ctx context.Context, reference storages.StoredFileReference, content io.Reader) error {
	receipt, err := a.storagesModule.WriteFile(ctx, reference, content)
	if err != nil {
		return err
	}
	return a.transactor.Transaction(ctx, func(txCtx context.Context) error {
		return a.storagesModule.ConfirmFileWrites(txCtx, []storages.WriteReceipt{receipt})
	})
}

func (a *App) NewJobRunner(names []string) *background.Runner {
	jobs := make([]background.Job, 0, len(names))
	for _, job := range a.jobs {
		for _, name := range names {
			if job.Name == name {
				jobs = append(jobs, job)
			}
		}
	}
	return background.NewRunner(background.RunnerDeps{Logger: a.logger, Clock: a.environment.Clock}, jobs...)
}

func (a *App) GetPostgresqlLogicalVersions(ctx context.Context, databaseIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	return a.databasesModule.GetPostgresqlLogicalVersions(ctx, databaseIDs)
}

func (a *App) GetLatestCompletedLogicalBackups(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]backups.LogicalBackup, error) {
	return a.backupsModule.GetLatestCompletedLogicalBackups(ctx, databaseIDs)
}
