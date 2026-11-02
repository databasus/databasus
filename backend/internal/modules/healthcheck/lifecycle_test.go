package healthcheck_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthcheckLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("copies all settings without attempts when a database is copied", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		team.pass(t)

		input := configInput(team.getConfig(t))
		input.IsHealthcheckEnabled = false
		input.IsNotificationEnabled = false
		input.IntervalMinutes = 10
		input.FailedAttemptsBeforeUnavailable = 2
		input.AttemptRetentionDays = 30
		team.saveConfig(t, input)

		copiedDatabase, err := team.Owner.Client.CopyDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.NotNil(t, copiedDatabase.JSON201, string(copiedDatabase.Body))
		copiedTeam := healthcheckTeam{Team: team.Team, Database: *copiedDatabase.JSON201}

		assert.Equal(t, input, configInput(copiedTeam.getConfig(t)))
		assert.Empty(t, copiedTeam.getAttempts(t))
	})
	t.Run("deletes the config and attempts in the database deletion transaction", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		team.pass(t)
		response, err := team.Owner.Client.DeleteDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))

		for _, table := range []string{"healthcheck_configs", "healthcheck_attempts"} {
			var count int64
			team.App.QuerySQL(t, &count, "SELECT count(*) FROM "+table+" WHERE database_id = ?", team.Database.Id)
			assert.Zero(t, count)
		}
	})
	t.Run("keeps config and attempts when a database deletion rolls back", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		team.pass(t)
		original := team.getConfig(t)

		team.App.ExecuteSQL(
			t,
			`CREATE FUNCTION refuse_healthcheck_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'delete refused'; END $$`,
		)
		team.App.ExecuteSQL(
			t,
			`CREATE TRIGGER refuse_healthcheck_delete BEFORE DELETE ON databases FOR EACH ROW EXECUTE FUNCTION refuse_healthcheck_delete()`,
		)

		response, err := team.Owner.Client.DeleteDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode())
		assert.Equal(t, original, team.getConfig(t))
		assert.Len(t, team.getAttempts(t), 1)
	})
}
