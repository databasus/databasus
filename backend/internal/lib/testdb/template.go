package testdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// One key for every template build on the server: builds are rare, and two worktrees building the
// same template must not both migrate it.
const templateBuildLockKey = 945_100_000

// hashMigrations covers the names and contents of the migrations, so a renamed, edited or added
// migration gets a template of its own.
func hashMigrations(files fs.FS) (string, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return "", fmt.Errorf("read the migrations: %w", err)
	}

	migrationNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			migrationNames = append(migrationNames, entry.Name())
		}
	}

	slices.Sort(migrationNames)

	hash := sha256.New()

	for _, migrationName := range migrationNames {
		content, err := fs.ReadFile(files, migrationName)
		if err != nil {
			return "", fmt.Errorf("read the migration %s: %w", migrationName, err)
		}

		_, _ = hash.Write([]byte(migrationName + "\x00"))
		_, _ = hash.Write(content)
		_, _ = hash.Write([]byte{0})
	}

	return hex.EncodeToString(hash.Sum(nil))[:16], nil
}

// buildTemplate migrates a database under a temporary name and renames it only after every
// migration succeeded, so an interrupted build never leaves a half-migrated template behind.
func buildTemplate(ctx context.Context, systemDb *gorm.DB, testDsn string, files fs.FS, templateName string) error {
	return systemDb.WithContext(ctx).Connection(func(lockSession *gorm.DB) error {
		err := lockSession.Exec("SELECT pg_advisory_lock(?)", templateBuildLockKey).Error
		if err != nil {
			return fmt.Errorf("lock the template build: %w", err)
		}

		defer func() {
			_ = lockSession.Exec("SELECT pg_advisory_unlock(?)", templateBuildLockKey).Error
		}()

		isBuilt, err := hasDatabase(lockSession, templateName)
		if err != nil || isBuilt {
			return err
		}

		buildingName := templateName + "_building"

		err = lockSession.Exec("DROP DATABASE IF EXISTS " + buildingName + " WITH (FORCE)").Error
		if err != nil {
			return fmt.Errorf("drop the unfinished template %s: %w", buildingName, err)
		}

		err = lockSession.Exec("CREATE DATABASE " + buildingName).Error
		if err != nil {
			return fmt.Errorf("create the template %s: %w", buildingName, err)
		}

		err = migrateDatabase(ctx, replaceDatabaseName(testDsn, buildingName), files)
		if err != nil {
			return err
		}

		err = lockSession.Exec("ALTER DATABASE " + buildingName + " RENAME TO " + templateName).Error
		if err != nil {
			return fmt.Errorf("publish the template %s: %w", templateName, err)
		}

		return nil
	})
}

func migrateDatabase(ctx context.Context, databaseDsn string, files fs.FS) error {
	db, err := database.Open(ctx, databaseDsn)
	if err != nil {
		return fmt.Errorf("connect to the template: %w", err)
	}

	_, migrateErr := database.Migrate(ctx, db, files)

	sqlDB, err := db.DB()
	if err == nil {
		err = sqlDB.Close()
	}

	return errors.Join(migrateErr, err)
}

func hasDatabase(session *gorm.DB, databaseName string) (bool, error) {
	var databaseCount int64

	err := session.Raw("SELECT count(*) FROM pg_database WHERE datname = ?", databaseName).Scan(&databaseCount).Error
	if err != nil {
		return false, fmt.Errorf("look up the database %s: %w", databaseName, err)
	}

	return databaseCount > 0, nil
}
