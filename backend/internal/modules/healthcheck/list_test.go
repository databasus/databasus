package healthcheck_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/database"
)

func TestHealthcheckAttemptList(t *testing.T) {
	t.Parallel()

	t.Run("filters since inclusively and pages newest first including tied timestamps", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		now := team.App.Clock.Now()
		attemptIDs := make([]uuid.UUID, 5)

		for index := range attemptIDs {
			attemptIDs[index] = uuid.New()
			team.App.ExecuteSQL(
				t,
				"INSERT INTO healthcheck_attempts (id,database_id,status,created_at) VALUES (?,?,'AVAILABLE',?)",
				attemptIDs[index],
				team.Database.Id,
				now.Add(time.Duration(index-4)*time.Hour),
			)
		}

		team.App.ExecuteSQL(t, "UPDATE healthcheck_attempts SET created_at = ? WHERE id = ?", now, attemptIDs[3])
		params := apiclient.ListHealthcheckAttemptsParams{
			DatabaseId: team.Database.Id,
			Since:      now.Add(-4 * time.Hour),
			Limit:      new(2),
		}

		listed := []apiclient.HealthcheckAttempt{}
		for {
			response, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(t.Context(), &params)
			require.NoError(t, err)
			require.NotNil(t, response.JSON200, string(response.Body))
			require.LessOrEqual(t, len(response.JSON200.Items), 2)
			require.NotEmpty(t, response.JSON200.Items)
			listed = append(listed, response.JSON200.Items...)
			if response.JSON200.NextCursor == nil {
				break
			}
			params.After = response.JSON200.NextCursor
			require.Less(t, len(listed), 7)
		}

		require.Len(t, listed, 5)
		ids := map[uuid.UUID]bool{}
		for index, attempt := range listed {
			assert.False(t, ids[attempt.Id])
			ids[attempt.Id] = true
			if index > 0 {
				previous := listed[index-1]
				assert.True(
					t,
					previous.CreatedAt.After(attempt.CreatedAt) ||
						previous.CreatedAt.Equal(attempt.CreatedAt) && previous.Id.String() > attempt.Id.String(),
				)
			}
		}

		params.Since = now.Add(-time.Hour)
		params.Limit = nil
		params.After = nil
		response, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))
		assert.Len(t, response.JSON200.Items, 2)
	})
	t.Run("returns an empty final page when the database has never been checked", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		assert.Empty(t, team.getAttempts(t))
	})
	t.Run("rejects a request with an invalid page size or cursor", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		missingCursor := (database.TimeCursor{Time: team.App.Clock.Now(), ID: uuid.New()}).Encode()

		for _, params := range []apiclient.ListHealthcheckAttemptsParams{
			{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC(), After: new("foreign")},
			{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC(), After: &missingCursor},
			{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC(), Limit: new(0)},
			{DatabaseId: team.Database.Id, Since: time.Unix(0, 0).UTC(), Limit: new(1441)},
		} {
			response, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(t.Context(), &params)
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
		}

		response, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(
			t.Context(),
			&apiclient.ListHealthcheckAttemptsParams{DatabaseId: team.Database.Id},
			func(_ context.Context, request *http.Request) error {
				query := request.URL.Query()
				query.Del("since")
				request.URL.RawQuery = query.Encode()
				return nil
			},
		)
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
	})
	t.Run("rejects a cursor when it belongs to another database", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		copiedDatabase, err := team.Owner.Client.CopyDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.NotNil(t, copiedDatabase.JSON201, string(copiedDatabase.Body))

		team.pass(t)
		team.App.Clock.Advance(time.Minute)
		team.pass(t)

		params := apiclient.ListHealthcheckAttemptsParams{
			DatabaseId: copiedDatabase.JSON201.Id,
			Since:      time.Unix(0, 0).UTC(),
			Limit:      new(1),
			After:      nil,
		}
		copiedAttempts, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		require.NotNil(t, copiedAttempts.JSON200, string(copiedAttempts.Body))
		require.NotNil(t, copiedAttempts.JSON200.NextCursor)

		params.After = copiedAttempts.JSON200.NextCursor
		nextCopiedAttempts, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		require.NotNil(t, nextCopiedAttempts.JSON200, string(nextCopiedAttempts.Body))
		require.Len(t, nextCopiedAttempts.JSON200.Items, 1)

		params.DatabaseId = team.Database.Id
		originalAttempts, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		requireProblem(t, problemCheck{
			Status:     originalAttempts.StatusCode(),
			Body:       originalAttempts.Body,
			HTTPStatus: http.StatusBadRequest,
			Code:       apiclient.ProblemCodeInvalidRequest,
		})
	})
}
