package healthcheck_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestHealthcheckAccess(t *testing.T) {
	t.Parallel()

	t.Run("allows reading but refuses saving when a viewer calls", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		viewer := team.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		original := team.getConfig(t)

		read, err := viewer.Client.GetHealthcheckConfigWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.NotNil(t, read.JSON200, string(read.Body))
		list, err := viewer.Client.ListHealthcheckAttemptsWithResponse(
			t.Context(),
			&apiclient.ListHealthcheckAttemptsParams{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC()},
		)
		require.NoError(t, err)
		require.NotNil(t, list.JSON200, string(list.Body))

		input := configInput(original)
		input.IsHealthcheckEnabled = false
		save, err := viewer.Client.UpdateHealthcheckConfigWithResponse(t.Context(), team.Database.Id, input)
		require.NoError(t, err)
		requireProblem(
			t,
			problemCheck{
				Status:     save.StatusCode(),
				Body:       save.Body,
				HTTPStatus: http.StatusForbidden,
				Code:       apiclient.ProblemCodePermissionDenied,
			},
		)

		assert.Equal(t, original, team.getConfig(t))
	})
	t.Run(
		"hides every operation with the same not_found body as a missing database when a stranger calls",
		func(t *testing.T) {
			t.Parallel()

			team := newHealthcheckTeam(t)
			stranger := team.App.SignUp(t, apptest.NewTestEmail("stranger"))
			original := team.getConfig(t)

			read, err := stranger.Client.GetHealthcheckConfigWithResponse(t.Context(), team.Database.Id)
			require.NoError(t, err)
			missing, err := stranger.Client.GetHealthcheckConfigWithResponse(t.Context(), uuid.New())
			require.NoError(t, err)
			requireProblem(
				t,
				problemCheck{
					Status:     read.StatusCode(),
					Body:       read.Body,
					HTTPStatus: http.StatusNotFound,
					Code:       apiclient.ProblemCodeNotFound,
				},
			)
			assert.JSONEq(t, string(missing.Body), string(read.Body))

			input := configInput(original)
			input.IntervalMinutes = 10
			save, err := stranger.Client.UpdateHealthcheckConfigWithResponse(
				t.Context(),
				team.Database.Id,
				input,
			)
			require.NoError(t, err)
			requireProblem(
				t,
				problemCheck{
					Status:     save.StatusCode(),
					Body:       save.Body,
					HTTPStatus: http.StatusNotFound,
					Code:       apiclient.ProblemCodeNotFound,
				},
			)

			list, err := stranger.Client.ListHealthcheckAttemptsWithResponse(
				t.Context(),
				&apiclient.ListHealthcheckAttemptsParams{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC()},
			)
			require.NoError(t, err)
			requireProblem(
				t,
				problemCheck{
					Status:     list.StatusCode(),
					Body:       list.Body,
					HTTPStatus: http.StatusNotFound,
					Code:       apiclient.ProblemCodeNotFound,
				},
			)

			assert.Equal(t, original, team.getConfig(t))
		},
	)
	t.Run("allows saving outside the workspace when an instance admin calls", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		input := configInput(team.getConfig(t))
		input.IntervalMinutes = 10
		save, err := team.Admin.Client.UpdateHealthcheckConfigWithResponse(t.Context(), team.Database.Id, input)
		require.NoError(t, err)
		require.NotNil(t, save.JSON200, string(save.Body))

		read, err := team.Admin.Client.GetHealthcheckConfigWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.NotNil(t, read.JSON200, string(read.Body))
		list, err := team.Admin.Client.ListHealthcheckAttemptsWithResponse(
			t.Context(),
			&apiclient.ListHealthcheckAttemptsParams{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC()},
		)
		require.NoError(t, err)
		require.NotNil(t, list.JSON200, string(list.Body))

		assert.Equal(t, input, configInput(team.getConfig(t)))
	})
	t.Run("requires a session for every operation", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		client := team.App.NewClient(t)

		read, err := client.GetHealthcheckConfigWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			problemCheck{
				Status:     read.StatusCode(),
				Body:       read.Body,
				HTTPStatus: http.StatusUnauthorized,
				Code:       apiclient.ProblemCodeUnauthenticated,
			},
		)

		save, err := client.UpdateHealthcheckConfigWithResponse(
			t.Context(),
			team.Database.Id,
			configInput(team.getConfig(t)),
		)
		require.NoError(t, err)
		requireProblem(
			t,
			problemCheck{
				Status:     save.StatusCode(),
				Body:       save.Body,
				HTTPStatus: http.StatusUnauthorized,
				Code:       apiclient.ProblemCodeUnauthenticated,
			},
		)

		list, err := client.ListHealthcheckAttemptsWithResponse(
			t.Context(),
			&apiclient.ListHealthcheckAttemptsParams{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC()},
		)
		require.NoError(t, err)
		requireProblem(
			t,
			problemCheck{
				Status:     list.StatusCode(),
				Body:       list.Body,
				HTTPStatus: http.StatusUnauthorized,
				Code:       apiclient.ProblemCodeUnauthenticated,
			},
		)
	})
}
