package catalog_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestTimelineHistoryForkPoint(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name    string
		columns string
		values  string
	}{
		{name: "refuses a parent timeline without a fork point", columns: "parent_timeline_id", values: "1"},
		{name: "refuses a fork point without a parent timeline", columns: "fork_lsn", values: "'0/3000148'"},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			catalog := newCatalog(t)

			err := catalog.App.ExecuteRefusedSQL(
				t,
				"INSERT INTO backups_physical_wal_history_files "+
					"(id, database_id, storage_id, timeline_id, file_name, history_filename, created_at, "+
					scenario.columns+") VALUES (?, ?, ?, 2, 'history', '00000002.history', now(), "+
					scenario.values+")",
				uuid.New(),
				catalog.DatabaseID,
				catalog.StorageID,
			)

			var postgresError *pgconn.PgError
			require.ErrorAs(t, err, &postgresError, "the insert must be refused")
			require.Equal(t, "23514", postgresError.Code)
			require.Equal(t, "ck_backups_physical_wal_history_files_fork", postgresError.ConstraintName)
		})
	}
}
