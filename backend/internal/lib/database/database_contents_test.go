package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestListPopulatedTables(t *testing.T) {
	t.Parallel()

	t.Run("lists tables with rows in every schema and leaves out the migration history", func(t *testing.T) {
		t.Parallel()
		db, err := database.Open(t.Context(), testdb.Open(t))
		require.NoError(t, err)
		t.Cleanup(func() { closePool(db) })
		require.NoError(
			t,
			db.Exec(`INSERT INTO workspaces (id, name, created_at) VALUES (gen_random_uuid(), 'w', now())`).Error,
		)
		require.NoError(t, db.Exec(`CREATE SCHEMA operator_notes`).Error)
		require.NoError(t, db.Exec(`CREATE TABLE operator_notes.notes (text TEXT)`).Error)
		require.NoError(t, db.Exec(`INSERT INTO operator_notes.notes VALUES ('kept')`).Error)

		populatedTables, err := database.ListPopulatedTables(t.Context(), db)

		require.NoError(t, err)
		assert.Contains(t, populatedTables, "public.workspaces")
		assert.Contains(t, populatedTables, "operator_notes.notes")
		assert.NotContains(t, populatedTables, "public.goose_db_version")
		assert.NotContains(t, populatedTables, "public.users")
	})
}

func TestReadRecordedMigrationVersions(t *testing.T) {
	t.Parallel()

	t.Run("reads no versions from a database that was never migrated and creates no table", func(t *testing.T) {
		t.Parallel()
		db, err := database.Open(t.Context(), testdb.OpenEmpty(t))
		require.NoError(t, err)
		t.Cleanup(func() { closePool(db) })

		recordedVersions, err := database.ReadRecordedMigrationVersions(t.Context(), db)
		require.NoError(t, err)
		isHistoryPresent, err := database.HasTable(t.Context(), db, "public.goose_db_version")
		require.NoError(t, err)

		assert.Empty(t, recordedVersions)
		assert.False(t, isHistoryPresent)
	})
}

func TestCheckNotNewer(t *testing.T) {
	t.Parallel()

	bundle := database.MigrationBundle{Digest: "bundle", Versions: []int64{20, 30}}

	t.Run("accepts no versions, older ones and the newest carried one", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, bundle.CheckNotNewer(nil))
		require.NoError(t, bundle.CheckNotNewer([]int64{10, 30}))
	})

	t.Run("refuses a version newer than the bundle with both versions in the message", func(t *testing.T) {
		t.Parallel()

		err := bundle.CheckNotNewer([]int64{20, 40})

		require.ErrorIs(t, err, database.ErrSchemaNewerThanBinary)
		assert.Contains(t, err.Error(), "40")
		assert.Contains(t, err.Error(), "30")
	})
}
