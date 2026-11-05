package catalog_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestartPhysicalCleanup(t *testing.T) {
	t.Parallel()

	t.Run("drops the orphan backup slot after failing a backup left running by a crash", func(t *testing.T) {
		t.Parallel()

		catalog := newConnectedCatalog(t)
		catalog.seedRunningBackupAndStreamer(t)

		database, err := catalog.Owner.Client.GetDatabaseWithResponse(t.Context(), catalog.DatabaseID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, database.StatusCode())
		settings, err := database.JSON200.Settings.AsPostgresqlPhysicalSettings()
		require.NoError(t, err)

		connection, err := pgx.Connect(
			t.Context(),
			fmt.Sprintf(
				"postgres://testuser:testpassword@%s:%d/postgres?sslmode=disable",
				settings.Host,
				settings.Port,
			),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = connection.Close(t.Context()) })

		var slotName string
		catalog.App.QuerySQL(
			t,
			&slotName,
			"SELECT backup_slot_name FROM databases_postgresql_physical WHERE database_id=?",
			catalog.DatabaseID,
		)
		_, err = connection.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1,true)", slotName)
		require.NoError(t, err)

		catalog.App.RestartModules(t)

		var slots int
		require.NoError(
			t,
			connection.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
				Scan(&slots),
		)

		assert.Zero(t, slots)
		assert.Nil(t, catalog.getRunningBackupID(t))
	})
}
