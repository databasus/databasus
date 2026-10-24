package database_test

import (
	"sync"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/migrations"
)

const (
	firstMigrationName  = "00001_create_first_table.sql"
	secondMigrationName = "00002_create_second_table.sql"
	thirdMigrationName  = "00003_create_third_table.sql"
)

func TestMigrate(t *testing.T) {
	t.Parallel()

	t.Run("applies every migration to an empty database and nothing on the second call", func(t *testing.T) {
		t.Parallel()
		db := openTestDatabase(t)

		firstApplied, err := database.Migrate(t.Context(), db, migrations.Files)
		require.NoError(t, err)

		secondApplied, err := database.Migrate(t.Context(), db, migrations.Files)
		require.NoError(t, err)

		embeddedMigrations, err := migrations.Files.ReadDir(".")
		require.NoError(t, err)
		assert.Len(t, firstApplied, len(embeddedMigrations))
		assert.Empty(t, secondApplied)
	})

	t.Run("applies a missed migration numbered below the latest applied one", func(t *testing.T) {
		t.Parallel()
		db := openTestDatabase(t)

		_, err := database.Migrate(t.Context(), db, fstest.MapFS{
			firstMigrationName: newMigration("CREATE TABLE first_table (id INT);"),
			thirdMigrationName: newMigration("CREATE TABLE third_table (id INT);"),
		})
		require.NoError(t, err)

		appliedNames, err := database.Migrate(t.Context(), db, fstest.MapFS{
			firstMigrationName:  newMigration("CREATE TABLE first_table (id INT);"),
			secondMigrationName: newMigration("CREATE TABLE second_table (id INT);"),
			thirdMigrationName:  newMigration("CREATE TABLE third_table (id INT);"),
		})

		require.NoError(t, err)
		assert.Equal(t, []string{secondMigrationName}, appliedNames)
		assert.True(t, hasTable(t, db, "second_table"))
	})

	t.Run("names the failed migration and leaves none of its changes", func(t *testing.T) {
		t.Parallel()
		db := openTestDatabase(t)

		_, err := database.Migrate(t.Context(), db, fstest.MapFS{
			firstMigrationName: newMigration("CREATE TABLE first_table (id INT);"),
			secondMigrationName: newMigration(
				"CREATE TABLE half_created_table (id INT);\nALTER TABLE first_table ADD COLUMN half_added INT;\nSELECT 1/0;",
			),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), secondMigrationName)
		assert.Equal(t, []int64{0, 1}, getAppliedVersions(t, db))
		assert.False(t, hasTable(t, db, "half_created_table"))
		assert.False(t, hasColumn(t, db, "first_table", "half_added"))
	})

	t.Run("applies a fixed migration after it failed", func(t *testing.T) {
		t.Parallel()
		db := openTestDatabase(t)

		_, err := database.Migrate(t.Context(), db, fstest.MapFS{
			firstMigrationName:  newMigration("CREATE TABLE first_table (id INT);"),
			secondMigrationName: newMigration("CREATE TABLE second_table (id INT);\nSELECT 1/0;"),
		})
		require.Error(t, err)

		appliedNames, err := database.Migrate(t.Context(), db, fstest.MapFS{
			firstMigrationName:  newMigration("CREATE TABLE first_table (id INT);"),
			secondMigrationName: newMigration("CREATE TABLE second_table (id INT);"),
		})

		require.NoError(t, err)
		assert.Equal(t, []string{secondMigrationName}, appliedNames)
		assert.True(t, hasTable(t, db, "second_table"))
	})

	t.Run("records each migration once when two calls migrate one database at the same time", func(t *testing.T) {
		t.Parallel()
		db := openTestDatabase(t)

		var waitGroup sync.WaitGroup

		migrateErrors := make([]error, 2)
		for callIndex := range migrateErrors {
			waitGroup.Go(func() {
				_, migrateErrors[callIndex] = database.Migrate(t.Context(), db, migrations.Files)
			})
		}

		waitGroup.Wait()

		for _, migrateError := range migrateErrors {
			require.NoError(t, migrateError)
		}

		var duplicateVersionCount int64
		require.NoError(t, db.WithContext(t.Context()).Raw(
			`SELECT COUNT(*) FROM (
				SELECT version_id FROM goose_db_version GROUP BY version_id HAVING COUNT(*) > 1
			) AS duplicates`,
		).Scan(&duplicateVersionCount).Error)
		assert.Zero(t, duplicateVersionCount)
	})
}

func newMigration(upStatements string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("-- +goose Up\n" + upStatements + "\n")}
}

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), testdb.CreateDatabase(t))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func getAppliedVersions(t *testing.T, db *gorm.DB) []int64 {
	t.Helper()

	var appliedVersions []int64
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY version_id",
	).Scan(&appliedVersions).Error)

	return appliedVersions
}

func hasTable(t *testing.T, db *gorm.DB, tableName string) bool {
	t.Helper()

	var isPresent bool
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = ?)", tableName,
	).Scan(&isPresent).Error)

	return isPresent
}

func hasColumn(t *testing.T, db *gorm.DB, tableName, columnName string) bool {
	t.Helper()

	var isPresent bool
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = ? AND column_name = ?)",
		tableName, columnName,
	).Scan(&isPresent).Error)

	return isPresent
}
