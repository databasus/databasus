package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoricalGoogleDriveMigration(t *testing.T) {
	t.Parallel()

	t.Run("preserves encrypted credentials when the historical storage rename runs", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, storagesMigrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := insertOldStorage(t, db, "GOOGLE_DRIVE", &workspaceID)
		execute(t, db, `INSERT INTO google_drive_storages (storage_id, client_id, client_secret, token_json)
			VALUES (?, 'synthetic-client', 'enc:synthetic-secret', 'enc:synthetic-token')`, storageID)

		applyMigrationsBefore(t, db, "20261026120001")

		var hasUnchangedSecrets bool
		require.NoError(t, db.WithContext(t.Context()).Raw(
			`SELECT client_secret = 'enc:synthetic-secret' AND token_json = 'enc:synthetic-token'
			 FROM storages_google_drive WHERE storage_id = ?`, storageID,
		).Scan(&hasUnchangedSecrets).Error)
		assert.True(t, hasUnchangedSecrets)
	})
}
