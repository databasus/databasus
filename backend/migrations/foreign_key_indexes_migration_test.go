package migrations_test

import (
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/migrations"
)

const (
	foreignKeyIndexesMigrationName    = "20261103172154_add_foreign_key_indexes.sql"
	foreignKeyIndexesMigrationVersion = 20261103172154
)

type foreignKeyColumn struct {
	table  string
	column string
}

func TestForeignKeyIndexesMigration(t *testing.T) {
	t.Parallel()

	t.Run("creates the foreign key indexes after the upgrade and drops them after the rollback", func(t *testing.T) {
		t.Parallel()
		foreignKeyColumns := []foreignKeyColumn{
			{table: "backups_physical", column: "storage_id"},
			{table: "backups_physical_wal_segments", column: "storage_id"},
			{table: "backups_physical_wal_history_files", column: "storage_id"},
			{table: "databases_notifiers", column: "notifier_id"},
		}
		db := openDatabaseMigratedBefore(t, foreignKeyIndexesMigrationName)

		applyAllMigrations(t, db, foreignKeyIndexesMigrationName)

		for _, foreignKey := range foreignKeyColumns {
			assert.NotEmpty(t, listIndexesLeadingWithColumn(t, db, foreignKey), foreignKey.table)
		}

		sqlDB, err := db.DB()
		require.NoError(t, err)
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			sqlDB,
			migrations.Files,
			goose.WithDisableGlobalRegistry(true),
		)
		require.NoError(t, err)

		_, err = provider.ApplyVersion(t.Context(), foreignKeyIndexesMigrationVersion, false)
		require.NoError(t, err)

		for _, foreignKey := range foreignKeyColumns {
			assert.Empty(t, listIndexesLeadingWithColumn(t, db, foreignKey), foreignKey.table)
		}
	})
}

func listIndexesLeadingWithColumn(t *testing.T, db *gorm.DB, foreignKey foreignKeyColumn) []string {
	t.Helper()

	return listStrings(t, db, `SELECT index_class.relname::text
		FROM pg_index AS table_index
		JOIN pg_class AS index_class ON index_class.oid = table_index.indexrelid
		JOIN pg_attribute AS leading_column
			ON leading_column.attrelid = table_index.indrelid AND leading_column.attnum = table_index.indkey[0]
		WHERE table_index.indrelid = ?::regclass AND leading_column.attname = ?`,
		foreignKey.table, foreignKey.column)
}
