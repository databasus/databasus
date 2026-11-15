package app

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/instancebackups"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification"
	"databasus-backend/migrations"
)

// LifecycleLock is held shared by the process while it uses the internal database.
type LifecycleLock = instancebackups.LifecycleLock

// OpenServerLifecycle runs before New for the server: it finishes an interrupted recovery
// publication and runs the startup recovery RECOVERY_DUMP_PATH and RECOVERY_MASTER_KEY_PATH
// configure, before any migration touches the database. The returned lock stays held until the
// server stops.
func OpenServerLifecycle(ctx context.Context, environment *Environment) (*instancebackups.LifecycleLock, error) {
	deps, err := newStartupDeps(environment)
	if err != nil {
		return nil, err
	}

	return instancebackups.OpenServerLifecycle(ctx, deps)
}

// PublishAcceptedRecovery runs after Serve returned because root admin accepted a recovery and
// after Close released the database connections; see instancebackups.PublishAcceptedRecovery.
func PublishAcceptedRecovery(
	ctx context.Context,
	environment *Environment,
	lock *instancebackups.LifecycleLock,
	isDrained bool,
) error {
	deps, err := newStartupDeps(environment)
	if err != nil {
		return err
	}

	return instancebackups.PublishAcceptedRecovery(ctx, deps, lock, isDrained)
}

// OpenConsoleLifecycle runs before New for a console command, which never runs a recovery.
func OpenConsoleLifecycle(environment *Environment) (*instancebackups.LifecycleLock, error) {
	deps, err := newStartupDeps(environment)
	if err != nil {
		return nil, err
	}

	return instancebackups.OpenConsoleLifecycle(deps)
}

func newStartupDeps(environment *Environment) (instancebackups.StartupDeps, error) {
	bundle, err := database.ReadMigrationBundle(migrations.Files)
	if err != nil {
		return instancebackups.StartupDeps{}, fmt.Errorf("read the migration bundle: %w", err)
	}

	return instancebackups.StartupDeps{
		DatabaseDsn:            environment.Config.DatabaseDsn,
		RecoveryStateDirectory: environment.Config.RecoveryStateDirectory,
		SecretKeyPath:          environment.Config.SecretKeyPath,
		DataFolder:             environment.Config.DataFolder,
		ClientToolsDir:         environment.ClientToolsDir,
		Release:                environment.Config.AppVersion,
		Bundle:                 bundle,
		DumpPath:               environment.Config.RecoveryDumpPath,
		MasterKeyPath:          environment.Config.RecoveryMasterKeyPath,
		NewerSchemaGuidance:    olderVersionRefusal,
		Normalizers:            getRecoveredStateNormalizers(),
		Clock:                  environment.Clock,
		Logger:                 environment.Logging.GetLogger(),
		InspectDisk:            environment.InspectDisk,
	}, nil
}

// getRecoveredStateNormalizers lists every module whose tables hold work or credentials of the lost
// instance, so a recovered snapshot starts with none of it.
func getRecoveredStateNormalizers() []instancebackups.RecoveredStateNormalizer {
	return []instancebackups.RecoveredStateNormalizer{
		func(ctx context.Context, db *gorm.DB, _ time.Time) (int, error) {
			return 0, users.NormalizeRecoveredState(ctx, db)
		},
		func(ctx context.Context, db *gorm.DB, _ time.Time) (int, error) {
			return storages.NormalizeRecoveredState(ctx, db)
		},
		func(ctx context.Context, db *gorm.DB, now time.Time) (int, error) {
			return 0, backups.NormalizeRecoveredState(ctx, db, now)
		},
		func(ctx context.Context, db *gorm.DB, now time.Time) (int, error) {
			return 0, verification.NormalizeRecoveredState(ctx, db, now)
		},
	}
}
