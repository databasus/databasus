package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	old_config "databasus-backend/internal/config"
	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/storage"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/migrations"
)

const olderVersionRefusal = "Returning to an older Databasus version after an upgrade is not supported. " +
	"Start the image of the newer version instead"

type App struct {
	environment *Environment
	db          *gorm.DB
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

	err = secrets.GetSecretKeyService().MigrateKeyFromDbToFileIfExist()
	if err != nil {
		return nil, fmt.Errorf("migrate the secret key from the database to its file: %w", err)
	}

	setUpDependencies()

	return &App{
		environment:     environment,
		db:              db,
		logger:          appLogger,
		backgroundGroup: nil,
	}, nil
}

func closeDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}
