package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"gorm.io/gorm"

	old_config "databasus-backend/internal/config"
	"databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/caching"
	"databasus-backend/internal/features/email"
	users_bridge "databasus-backend/internal/features/users"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/migrations"
)

// The listeners of the old features live in process-wide state, so a second App in one process,
// such as the next test application, must not register them again.
var setUpDependenciesOnce sync.Once

const olderVersionRefusal = "Returning to an older Databasus version after an upgrade is not supported. " +
	"Start the image of the newer version instead"

type App struct {
	usersModule users.Module
	environment *Environment
	db          *gorm.DB
	transactor  *database.Transactor
	logger      *slog.Logger
	// backgroundGroup exists only while Serve runs, so a console command cannot start a loop.
	backgroundGroup *background.Group
	// runner is started only by Serve; a console command and a test application run its passes
	// through RunJobPass.
	runner *background.Runner
}

// New prepares everything the server and the console commands share: the schema of the current
// version, the data directories, the master key file and the wiring of the old code.
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

	err = files_utils.EnsureDirectories([]string{environment.Config.TempFolder, environment.Config.DataFolder})
	if err != nil {
		closeDatabase(db)

		return nil, fmt.Errorf("ensure data directories: %w", err)
	}

	old_config.Install(environment.Config)
	storage.Install(db)

	if err := parseModelSchemas(db); err != nil {
		closeDatabase(db)

		return nil, err
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

	encryption.Install(masterKey, crypto.NewFieldEncryptor(masterKey))
	email.Install(environment.Mailer)

	cacheStore := cache.NewMemoryStore(cache.Deps{
		PayloadBudgetBytes: cache.DefaultPayloadBudgetBytes,
		Clock:              environment.Clock,
	})
	rateLimiter := ratelimit.NewMemoryCounter(ratelimit.Deps{Store: cacheStore, Clock: environment.Clock})
	caching.Install(cacheStore, rateLimiter)

	auditLog := audit.New(audit.Deps{
		DB:          db,
		Clock:       environment.Clock,
		AuditLogger: environment.Logging.GetAuditLogger(),
	})
	audit_logs.InstallAuditLog(auditLog)

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
	users_bridge.Install(usersModule)

	setUpDependenciesOnce.Do(setUpDependencies)

	jobs := append([]background.Job{newAuditEventsCleanupJob(auditLog, appLogger)}, usersModule.GetJobs()...)
	runner := background.NewRunner(background.RunnerDeps{Logger: appLogger, Clock: environment.Clock}, jobs...)

	return &App{
		usersModule:     usersModule,
		environment:     environment,
		db:              db,
		transactor:      transactor,
		logger:          appLogger,
		backgroundGroup: nil,
		runner:          runner,
	}, nil
}

// RunJobPass runs one pass of the named background job and returns its error.
func (a *App) RunJobPass(ctx context.Context, name string) error {
	return a.runner.RunPass(ctx, name)
}

// moveLegacyAndOpenMasterKey runs before the old code is wired, so nothing encrypts with a key that a
// legacy key from the database is about to replace.
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
