package backups_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const restoreTargetPassword = "restore-secret-5d1e"

func getRestoreFields() []string {
	return []string{"id", "backupId", "status", "failMessage", "durationMs", "createdAt"}
}

//nolint:tparallel // The subtests share one source server.
func TestRestoreTargetCredentials(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns the restore without any field of its target", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		targetSpec := server.newTargetSpec(t)
		startedRestore := requireStartedRestore(t, tm.owner, database.backup.Id, server.newRestoreTarget(t, targetSpec))
		waitForRestoreStatus(t, tm.owner, startedRestore.Id, apiclient.RestoreStatusCOMPLETED)

		response, err := tm.owner.Client.GetRestoreWithResponse(t.Context(), startedRestore.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		var restoreBody map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(response.Body, &restoreBody))

		for field := range restoreBody {
			assert.True(t, slices.Contains(getRestoreFields(), field), "the restore returns the field %s", field)
		}

		assert.NotContains(t, string(response.Body), targetSpec.databaseName)
		assert.NotContains(t, string(response.Body), containers.PostgresPassword)
	})

	t.Run("stores and logs no target password after the restore completes", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		targetSpec := newTargetSpecOfOwnRole(t, server)

		startedRestore := requireStartedRestore(t, tm.owner, database.backup.Id, server.newRestoreTarget(t, targetSpec))
		waitForRestoreStatus(t, tm.owner, startedRestore.Id, apiclient.RestoreStatusCOMPLETED)
		waitForRestoreLogRecord(
			t,
			tm,
			restoreLogRecord{message: "logical restore completed", restoreID: startedRestore.Id},
		)

		assert.Empty(t, listTablesHolding(t, tm, restoreTargetPassword))
		assert.NotContains(t, readApplicationLogs(t, tm), restoreTargetPassword)
	})
}

// newTargetSpecOfOwnRole creates a role with restoreTargetPassword and a target database it owns.
func newTargetSpecOfOwnRole(t *testing.T, server restoreServer) restoreTargetSpec {
	t.Helper()

	roleName := "restorer_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	server.exec(t, containers.PostgresDatabase,
		"CREATE ROLE "+roleName+" LOGIN PASSWORD '"+restoreTargetPassword+"'")

	targetSpec := server.newTargetSpec(t)
	server.exec(t, containers.PostgresDatabase, "ALTER DATABASE "+targetSpec.databaseName+" OWNER TO "+roleName)
	targetSpec.username = roleName
	targetSpec.password = restoreTargetPassword

	return targetSpec
}

// listTablesHolding searches every row of every table of the internal database for the text.
func listTablesHolding(t *testing.T, tm team, text string) []string {
	t.Helper()

	var tables []struct {
		TableSchema string
		TableName   string
	}
	tm.app.QuerySQL(t, &tables, `SELECT table_schema, table_name FROM information_schema.tables
		WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema')`)
	require.NotEmpty(t, tables)

	var holdingTables []string

	for _, table := range tables {
		tableName := pgx.Identifier{table.TableSchema, table.TableName}.Sanitize()

		if countRows(t, tm, "SELECT COUNT(*) FROM "+tableName+" AS stored_row WHERE stored_row::text LIKE ?",
			"%"+text+"%") > 0 {
			holdingTables = append(holdingTables, tableName)
		}
	}

	return holdingTables
}
