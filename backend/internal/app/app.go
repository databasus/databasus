package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"gorm.io/gorm"

	old_config "databasus-backend/internal/config"
	"databasus-backend/internal/features/email"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
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
	environment *Environment
	db          *gorm.DB
	transactor  *database.Transactor
	logger      *slog.Logger
	// backgroundGroup exists only while Serve runs, so a console command cannot start a loop.
	backgroundGroup *background.Group
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
	email.Install(mailer.NewInstanceSender(mailer.Deps{
		Settings: environment.Config.GetInstanceMailSettings(),
		Logger:   appLogger,
	}))

	setUpDependenciesOnce.Do(setUpDependencies)

	return &App{
		environment:     environment,
		db:              db,
		transactor:      transactor,
		logger:          appLogger,
		backgroundGroup: nil,
	}, nil
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
