package databases_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestGetPostgresqlLogicalVersions(t *testing.T) {
	t.Parallel()

	t.Run(
		"returns versions without decrypting settings when IDs include other engines and missing databases",
		func(t *testing.T) {
			t.Parallel()

			postgresServer := containers.StartPostgres(t, "postgres:18")
			tm := newTeam(t)
			savedDatabase := tm.createDatabase(
				t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
			)
			mysqlID := tm.insertMysqlDatabase(t)
			tm.app.ExecuteSQL(
				t,
				"UPDATE databases_postgresql_logical SET password = ? WHERE database_id = ?",
				"unreadable-ciphertext",
				savedDatabase.Id,
			)

			versions := tm.app.GetPostgresqlLogicalVersions(t, []uuid.UUID{savedDatabase.Id, mysqlID, uuid.New()})
			assert.Equal(t, map[uuid.UUID]string{savedDatabase.Id: "18"}, versions)
			assert.Empty(t, tm.app.GetPostgresqlLogicalVersions(t, nil))
		},
	)
}
