package verification_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationAccess(t *testing.T) {
	t.Parallel()

	t.Run("allows reads and refuses edits when a workspace viewer calls", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		viewer := team.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		original := team.config(t)

		config, err := viewer.Client.GetVerificationConfigWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.NotNil(t, config.JSON200, string(config.Body))
		listed, err := viewer.Client.ListVerificationsWithResponse(
			t.Context(),
			&apiclient.ListVerificationsParams{DatabaseId: team.Database.Id, Limit: nil, After: nil},
		)
		require.NoError(t, err)
		require.NotNil(t, listed.JSON200, string(listed.Body))
		assert.Len(t, listed.JSON200.Items, 1)
		read, err := viewer.Client.GetVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		require.NotNil(t, read.JSON200, string(read.Body))

		changedConfig := configInput(original)
		changedConfig.IsAutomaticVerificationEnabled = !original.IsAutomaticVerificationEnabled
		saved, err := viewer.Client.UpdateVerificationConfigWithResponse(
			t.Context(),
			team.Database.Id,
			changedConfig,
		)
		require.NoError(t, err)
		requireProblem(t, saved.StatusCode(), saved.Body, http.StatusForbidden, apiclient.ProblemCodePermissionDenied)

		started, err := viewer.Client.StartVerificationWithResponse(
			t.Context(),
			apiclient.StartVerificationInput{BackupId: backup.Id},
		)
		require.NoError(t, err)
		requireProblem(
			t,
			started.StatusCode(),
			started.Body,
			http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied,
		)

		canceled, err := viewer.Client.CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			canceled.StatusCode(),
			canceled.Body,
			http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied,
		)

		assert.Equal(t, original, team.config(t))
		assert.Len(t, team.list(t), 1)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, verification.Id).Status)
	})
	t.Run("returns the same not_found body as missing objects when a stranger calls", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		stranger := team.App.SignUp(t, apptest.NewTestEmail("stranger"))
		original := team.config(t)

		config, err := stranger.Client.GetVerificationConfigWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		missingConfig, err := stranger.Client.GetVerificationConfigWithResponse(t.Context(), uuid.New())
		require.NoError(t, err)
		requireProblem(t, config.StatusCode(), config.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.JSONEq(t, string(config.Body), string(missingConfig.Body))

		changedConfig := configInput(original)
		changedConfig.IsAutomaticVerificationEnabled = !original.IsAutomaticVerificationEnabled
		saved, err := stranger.Client.UpdateVerificationConfigWithResponse(
			t.Context(),
			team.Database.Id,
			changedConfig,
		)
		require.NoError(t, err)
		requireProblem(t, saved.StatusCode(), saved.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)

		started, err := stranger.Client.StartVerificationWithResponse(
			t.Context(),
			apiclient.StartVerificationInput{BackupId: backup.Id},
		)
		require.NoError(t, err)
		missingStart, err := stranger.Client.StartVerificationWithResponse(
			t.Context(),
			apiclient.StartVerificationInput{BackupId: uuid.New()},
		)
		require.NoError(t, err)
		requireProblem(t, started.StatusCode(), started.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.JSONEq(t, string(started.Body), string(missingStart.Body))

		read, err := stranger.Client.GetVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		missingRead, err := stranger.Client.GetVerificationWithResponse(t.Context(), uuid.New())
		require.NoError(t, err)
		requireProblem(t, read.StatusCode(), read.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.JSONEq(t, string(read.Body), string(missingRead.Body))

		listed, err := stranger.Client.ListVerificationsWithResponse(
			t.Context(),
			&apiclient.ListVerificationsParams{DatabaseId: team.Database.Id, Limit: nil, After: nil},
		)
		require.NoError(t, err)
		requireProblem(t, listed.StatusCode(), listed.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		canceled, err := stranger.Client.CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		requireProblem(t, canceled.StatusCode(), canceled.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)

		assert.Equal(t, original, team.config(t))
		assert.Len(t, team.list(t), 1)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, verification.Id).Status)
	})
	t.Run("allows edits when an instance admin belongs to no workspace", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		saved, err := team.Admin.Client.UpdateVerificationConfigWithResponse(
			t.Context(),
			team.Database.Id,
			configInput(team.config(t)),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, saved.StatusCode())
		started, err := team.Admin.Client.StartVerificationWithResponse(
			t.Context(),
			apiclient.StartVerificationInput{BackupId: backup.Id},
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, started.StatusCode())
	})
	t.Run("requires a session for every workspace verification operation", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		client := team.App.NewClient(t)
		config, err := client.GetVerificationConfigWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			config.StatusCode(),
			config.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		saved, err := client.UpdateVerificationConfigWithResponse(
			t.Context(),
			team.Database.Id,
			configInput(team.config(t)),
		)
		require.NoError(t, err)
		requireProblem(t, saved.StatusCode(), saved.Body, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
		start, err := client.StartVerificationWithResponse(
			t.Context(),
			apiclient.StartVerificationInput{BackupId: backup.Id},
		)
		require.NoError(t, err)
		requireProblem(t, start.StatusCode(), start.Body, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)

		read, err := client.GetVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		requireProblem(t, read.StatusCode(), read.Body, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
		list, err := client.ListVerificationsWithResponse(
			t.Context(),
			&apiclient.ListVerificationsParams{DatabaseId: team.Database.Id, Limit: nil, After: nil},
		)
		require.NoError(t, err)
		requireProblem(t, list.StatusCode(), list.Body, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)

		cancel, err := client.CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			cancel.StatusCode(),
			cancel.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)
	})
}
