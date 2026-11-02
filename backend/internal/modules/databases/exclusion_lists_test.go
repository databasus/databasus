package databases_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one source server.
func TestDatabaseExclusionLists(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("stores trimmed unique names when the lists are pasted over several lines", func(t *testing.T) {
		tm := newTeam(t)
		createdDatabase := tm.createDatabaseNamed(t, "Orders", fromLogicalSettings(t, newLogicalSettings(server)))
		pastedLogical := getLogicalSettings(t, createdDatabase)
		pastedLogical.ExcludeTables = []string{"orders", "\npersonnel_real_time", " ", "orders"}
		pastedLogical.IncludeSchemas = []string{"public", "\treporting"}
		pastedLogical.ExcludeTableData = []string{"logs, history_*", "\nlogs", " "}
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, pastedLogical)

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		storedLogical := getLogicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id))
		assert.Equal(t, []string{"orders", "personnel_real_time"}, storedLogical.ExcludeTables)
		assert.Equal(t, []string{"public", "reporting"}, storedLogical.IncludeSchemas)
		assert.Equal(t, []string{"logs", "history_*"}, storedLogical.ExcludeTableData)
	})

	t.Run("keeps the excluded table data, copies it and clears it after a change without it", func(t *testing.T) {
		tm := newTeam(t)
		logical := newLogicalSettings(server)
		logical.ExcludeTables = []string{"audit_tmp"}
		logical.ExcludeTableData = []string{"logs", "history_*"}
		createdDatabase := tm.createDatabaseNamed(t, "Data excluded", fromLogicalSettings(t, logical))
		readLogical := getLogicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id))
		require.Equal(t, []string{"audit_tmp"}, readLogical.ExcludeTables)
		require.Equal(t, []string{"logs", "history_*"}, readLogical.ExcludeTableData)

		copiedDatabase := copyDatabase(t, tm.owner, createdDatabase.Id)
		readLogical.ExcludeTableData = []string{}
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, readLogical)
		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, []string{"logs", "history_*"},
			getLogicalSettings(t, getDatabase(t, tm.owner, copiedDatabase.Id)).ExcludeTableData)
		changedLogical := getLogicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id))
		assert.Empty(t, changedLogical.ExcludeTableData)
		assert.Equal(t, []string{"audit_tmp"}, changedLogical.ExcludeTables)
	})
}
