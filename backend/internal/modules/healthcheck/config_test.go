package healthcheck_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestHealthcheckConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns the enabled default after database creation without changing it on read", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		config := team.getConfig(t)
		assert.True(t, config.IsHealthcheckEnabled)
		assert.True(t, config.IsNotificationEnabled)
		assert.Equal(t, 1, config.IntervalMinutes)
		assert.Equal(t, 3, config.FailedAttemptsBeforeUnavailable)
		assert.Equal(t, 7, config.AttemptRetentionDays)
		assert.Equal(t, config, team.getConfig(t))
	})
	t.Run("persists all fields and an audit event when a member saves", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		member := team.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		input := apiclient.HealthcheckConfigInput{
			IsHealthcheckEnabled:            true,
			IsNotificationEnabled:           false,
			IntervalMinutes:                 5,
			FailedAttemptsBeforeUnavailable: 2,
			AttemptRetentionDays:            30,
		}

		response, err := member.Client.UpdateHealthcheckConfigWithResponse(t.Context(), team.Database.Id, input)
		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))
		assert.Equal(t, input, configInput(*response.JSON200))
		assert.Equal(t, input, configInput(team.getConfig(t)))

		events := apptest.ListAllAuditEvents(
			t,
			team.Owner,
			apiclient.ListAuditEventsParams{WorkspaceId: &team.Workspace.Id},
		)

		hasConfigEvent := false
		for _, event := range events {
			if event.Message == "Healthcheck config updated for database '"+team.Database.Name+"'" {
				hasConfigEvent = true
			}
		}
		assert.True(t, hasConfigEvent)
	})
	t.Run("keeps every field when a numeric value is below one", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		original := team.getConfig(t)
		for _, field := range []string{"interval", "threshold", "retention"} {
			input := configInput(original)

			switch field {
			case "interval":
				input.IntervalMinutes = 0
			case "threshold":
				input.FailedAttemptsBeforeUnavailable = 0
			case "retention":
				input.AttemptRetentionDays = 0
			}

			response, err := team.Owner.Client.UpdateHealthcheckConfigWithResponse(t.Context(), team.Database.Id, input)
			require.NoError(t, err)
			requireProblem(
				t,
				problemCheck{
					Status:     response.StatusCode(),
					Body:       response.Body,
					HTTPStatus: http.StatusBadRequest,
					Code:       apiclient.ProblemCodeInvalidRequest,
				},
			)
			assert.Equal(t, original, team.getConfig(t))
		}
	})
	t.Run("rolls back the config and status when the audit write fails", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		original := team.getConfig(t)
		team.App.ExecuteSQL(
			t,
			"ALTER TABLE lib_audit_events ADD CONSTRAINT ck_healthcheck_audit_failure CHECK (message NOT LIKE 'Healthcheck config%')",
		)

		input := configInput(original)
		input.IsHealthcheckEnabled = false
		response, err := team.Owner.Client.UpdateHealthcheckConfigWithResponse(t.Context(), team.Database.Id, input)
		require.NoError(t, err)
		requireProblem(
			t,
			problemCheck{
				Status:     response.StatusCode(),
				Body:       response.Body,
				HTTPStatus: http.StatusInternalServerError,
				Code:       apiclient.ProblemCodeInternal,
			},
		)

		assert.Equal(t, original, team.getConfig(t))
		assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), team.getHealthStatus(t))
	})
	t.Run("returns internal without creating a config when its row is missing", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		team.App.ExecuteSQL(t, "DELETE FROM healthcheck_configs WHERE database_id = ?", team.Database.Id)

		response, err := team.Owner.Client.GetHealthcheckConfigWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			problemCheck{
				Status:     response.StatusCode(),
				Body:       response.Body,
				HTTPStatus: http.StatusInternalServerError,
				Code:       apiclient.ProblemCodeInternal,
			},
		)

		var count int64
		team.App.QuerySQL(t, &count, "SELECT count(*) FROM healthcheck_configs WHERE database_id = ?", team.Database.Id)
		assert.Zero(t, count)
	})
}
