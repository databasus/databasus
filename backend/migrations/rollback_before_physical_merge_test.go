package migrations_test

import (
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"databasus-backend/migrations"
)

func TestRollbackBeforePhysicalMerge(t *testing.T) {
	t.Parallel()

	t.Run(
		"restores the released schema when the reversible refactoring migrations are rolled back",
		func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, mergePhysicalBackupsMigrationName)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			provider, err := goose.NewProvider(
				goose.DialectPostgres,
				sqlDB,
				migrations.Files,
				goose.WithDisableGlobalRegistry(true),
			)
			require.NoError(t, err)

			_, err = provider.DownTo(t.Context(), 20261022072202)

			require.NoError(t, err)
			_, err = provider.UpTo(t.Context(), 20261027214332)
			require.NoError(t, err)
		},
	)
}
