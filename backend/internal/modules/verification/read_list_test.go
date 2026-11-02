package verification_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestListVerifications(t *testing.T) {
	t.Parallel()

	t.Run("pages newest first without table statistics when a database has three verifications", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verifications := make([]apiclient.Verification, 3)
		for index := range verifications {
			verifications[index] = team.start(t, backup.Id)
			canceled, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verifications[index].Id)
			require.NoError(t, err)
			require.NotNil(t, canceled.JSON200, string(canceled.Body))
			team.App.Clock.Advance(time.Second)
		}

		first, err := team.Owner.Client.ListVerificationsWithResponse(
			t.Context(),
			&apiclient.ListVerificationsParams{DatabaseId: team.Database.Id, Limit: new(2), After: nil},
		)
		require.NoError(t, err)
		require.NotNil(t, first.JSON200, string(first.Body))
		require.Len(t, first.JSON200.Items, 2)
		assert.Equal(t, verifications[2].Id, first.JSON200.Items[0].Id)
		assert.Equal(t, verifications[1].Id, first.JSON200.Items[1].Id)
		require.NotNil(t, first.JSON200.NextCursor)
		assert.NotContains(t, string(first.Body), "tableStats")

		last, err := team.Owner.Client.ListVerificationsWithResponse(
			t.Context(),
			&apiclient.ListVerificationsParams{
				DatabaseId: team.Database.Id,
				Limit:      new(2),
				After:      first.JSON200.NextCursor,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, last.JSON200)
		require.Len(t, last.JSON200.Items, 1)
		assert.Equal(t, verifications[0].Id, last.JSON200.Items[0].Id)
		assert.Nil(t, last.JSON200.NextCursor)
	})
	t.Run("returns invalid_request when the cursor or page size is invalid", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		for _, params := range []apiclient.ListVerificationsParams{{DatabaseId: team.Database.Id, Limit: nil, After: new("foreign")}, {DatabaseId: team.Database.Id, Limit: new(0), After: nil}, {DatabaseId: team.Database.Id, Limit: new(101), After: nil}} {
			response, err := team.Owner.Client.ListVerificationsWithResponse(t.Context(), &params)
			require.NoError(t, err)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest,
			)
		}
	})
}
