package postgresql

import (
	"testing"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestBackupAndRestorePostgresqlThroughPgBouncer(t *testing.T) {
	t.Parallel()
	t.Run("restores schema and rows with four jobs through pgbouncer", func(t *testing.T) {
		t.Parallel()
		pooler := containers.StartPostgresThroughPgBouncer(t, "postgres:18")
		server := connectToPostgresql(t, pooler.Endpoint, postgresqlPlainConnection)
		tables := make([]string, 4)
		for tableIndex := range tables {
			tables[tableIndex] = server.createFilledTable(t)
		}

		settings := server.newSettings()
		settings.CpuCount = 4

		restored := server.backUpAndRestore(t, postgresqlCycle{
			settings: settings, encryption: apiclient.BackupEncryptionNONE,
			changeTarget:            func(target *apiclient.PostgresqlRestoreTarget) { target.CpuCount = 4 },
			prepareRestoredDatabase: nil,
		})

		restored.tm.RequireBackupSizeMatchesDownload(t, restored.backup)
		for _, tableName := range tables {
			verifyPostgresqlRows(t, server.admin, restored.restoredDB, tableName)

			schemaQuery := `SELECT column_name || ':' || data_type || ':' || is_nullable FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 ORDER BY ordinal_position`
			var sourceColumns, restoredColumns []string
			require.NoError(t, server.admin.SelectContext(t.Context(), &sourceColumns, schemaQuery, tableName))
			require.NoError(t, restored.restoredDB.SelectContext(t.Context(), &restoredColumns, schemaQuery, tableName))
			require.Equal(t, sourceColumns, restoredColumns)

			var primaryKey string
			require.NoError(
				t,
				restored.restoredDB.GetContext(
					t.Context(),
					&primaryKey,
					`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = $1::regclass AND contype = 'p'`,
					tableName,
				),
			)
			require.Equal(t, "PRIMARY KEY (id)", primaryKey)
		}
	})
}
