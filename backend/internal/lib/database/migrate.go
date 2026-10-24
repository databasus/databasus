package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"gorm.io/gorm"
)

// Migrate applies every migration in files that the database has not recorded yet, including one
// numbered below the latest applied version, because migrations merged in any order ship in one
// release. A PostgreSQL session lock holds off another process migrating the same database, such
// as a console command started while the server upgrades. Each migration runs in its own
// transaction, so a failed one leaves no partial changes behind. It returns the names of the
// migrations it applied.
//
// It applies nothing and returns ErrSchemaNewerThanBinary when the newest recorded migration is
// newer than the newest one in files. A recorded migration missing from files but older than that
// is no reason to stop: releases have shipped migrations that later releases deleted or renamed.
func Migrate(ctx context.Context, db *gorm.DB, files fs.FS) ([]string, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get the connection pool: %w", err)
	}

	sessionLocker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("create the migration lock: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		files,
		goose.WithAllowOutofOrder(true),
		goose.WithSessionLocker(sessionLocker),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return nil, fmt.Errorf("read the migrations: %w", err)
	}

	err = ensureSchemaNotNewer(ctx, provider)
	if err != nil {
		return nil, err
	}

	results, err := provider.Up(ctx)

	appliedNames := make([]string, 0, len(results))
	for _, result := range results {
		if result.Error == nil {
			appliedNames = append(appliedNames, filepath.Base(result.Source.Path))
		}
	}

	if err != nil {
		var partialError *goose.PartialError
		if errors.As(err, &partialError) && partialError.Failed != nil {
			return appliedNames, fmt.Errorf(
				"apply migration %s: %w", filepath.Base(partialError.Failed.Source.Path), partialError.Err,
			)
		}

		return appliedNames, fmt.Errorf("apply migrations: %w", err)
	}

	return appliedNames, nil
}

func ensureSchemaNotNewer(ctx context.Context, provider *goose.Provider) error {
	databaseVersion, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("read the newest applied migration: %w", err)
	}

	sources := provider.ListSources()
	binaryVersion := sources[len(sources)-1].Version

	if databaseVersion > binaryVersion {
		return fmt.Errorf(
			"%w: the newest migration in the database is %d, the newest one this version carries is %d",
			ErrSchemaNewerThanBinary, databaseVersion, binaryVersion,
		)
	}

	return nil
}
