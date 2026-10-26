package migrations_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/migrations"
)

func openDatabaseMigratedBefore(t *testing.T, migrationName string) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), testdb.OpenEmpty(t))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	migrationFiles, err := fs.ReadDir(migrations.Files, ".")
	require.NoError(t, err)

	earlierMigrations := fstest.MapFS{}
	for _, migrationFile := range migrationFiles {
		if !strings.HasSuffix(migrationFile.Name(), ".sql") || migrationFile.Name() >= migrationName {
			continue
		}

		content, err := fs.ReadFile(migrations.Files, migrationFile.Name())
		require.NoError(t, err)

		earlierMigrations[migrationFile.Name()] = &fstest.MapFile{Data: content}
	}

	_, err = database.Migrate(t.Context(), db, earlierMigrations)
	require.NoError(t, err)

	return db
}

func applyAllMigrations(t *testing.T, db *gorm.DB, migrationName string) {
	t.Helper()

	appliedNames, err := database.Migrate(t.Context(), db, migrations.Files)
	require.NoError(t, err)
	require.Contains(t, appliedNames, migrationName)
}

func insertWorkspace(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()

	workspaceID := uuid.New()
	execute(t, db, "INSERT INTO workspaces (id, name) VALUES (?, ?)", workspaceID, "workspace "+workspaceID.String())

	return workspaceID
}

func insertDatabase(t *testing.T, db *gorm.DB, workspaceID *uuid.UUID) uuid.UUID {
	t.Helper()

	databaseID := uuid.New()
	execute(t, db, "INSERT INTO databases (id, name, type, workspace_id) VALUES (?, 'payments', 'POSTGRES', ?)",
		databaseID, workspaceID)

	return databaseID
}

func listIDs(t *testing.T, db *gorm.DB, query string) []uuid.UUID {
	t.Helper()

	var ids []uuid.UUID
	require.NoError(t, db.WithContext(t.Context()).Raw(query).Scan(&ids).Error)

	return ids
}

func execute(t *testing.T, db *gorm.DB, statement string, values ...any) {
	t.Helper()

	require.NoError(t, db.WithContext(t.Context()).Exec(statement, values...).Error)
}
