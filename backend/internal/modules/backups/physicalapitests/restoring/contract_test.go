package restoring_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalRestoreTokenContract(t *testing.T) {
	t.Parallel()

	t.Run("expires after 15 minutes when a token has not been used", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := createStreamableFullBackup(t, catalog)
		var token restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &token,
			},
		)

		assert.Equal(t, catalog.App.Clock.Now().Add(15*time.Minute), token.ExpiresAt)
		catalog.App.Clock.Advance(15*time.Minute + time.Second)
		response := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusUnauthorized},
		)

		var problem apiclient.Problem
		require.NoError(t, json.Unmarshal(response.Body, &problem))
		assert.Equal(t, apiclient.ProblemCodeUnauthenticated, problem.Code)
	})
	t.Run("returns no_restorable_backup when the full backup is deleted after token issue", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := createStreamableFullBackup(t, catalog)
		var token restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &token,
			},
		)

		deletion, err := catalog.Owner.Client.DeletePhysicalBackupWithResponse(t.Context(), full.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deletion.StatusCode(), "%s", deletion.Body)
		response := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusConflict},
		)

		var problem apiclient.Problem
		require.NoError(t, json.Unmarshal(response.Body, &problem))
		assert.Equal(t, apiclient.ProblemCodeNoRestorableBackup, problem.Code)
	})
	t.Run("returns not_found when a backup token names a backup deleted after issue", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := createStreamableFullBackup(t, catalog)
		var token restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &token,
			},
		)

		deletion, err := catalog.Owner.Client.DeletePhysicalBackupWithResponse(t.Context(), full.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deletion.StatusCode(), "%s", deletion.Body)
		response := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusNotFound},
		)

		var problem apiclient.Problem
		require.NoError(t, json.Unmarshal(response.Body, &problem))
		assert.Equal(t, apiclient.ProblemCodeNotFound, problem.Code)
	})
	t.Run("spends the token when a competing restore stream refuses its first download", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := createStreamableFullBackup(t, catalog)
		var token restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &token,
			},
		)

		catalog.holdRestoreStream(t)

		response := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusConflict},
		)

		var problem apiclient.Problem
		require.NoError(t, json.Unmarshal(response.Body, &problem))
		assert.Equal(t, apiclient.ProblemCodeAlreadyRunning, problem.Code)

		catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusUnauthorized},
		)
	})
	t.Run(
		"returns the same public script without credentials when fetched with and without a session",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			full := createStreamableFullBackup(t, catalog)
			var token restoreTokenResponse
			catalog.call(
				t,
				catalogCall{
					Request:       apptest.PhysicalBackupTokenRequest(full.ID),
					Authorization: "Bearer " + catalog.Owner.Token,
					StatusCode:    http.StatusCreated,
					Destination:   &token,
				},
			)

			_, publicScript := catalog.App.CallPhysicalAPI(t, apptest.Session{}, apptest.PhysicalScriptRequest())
			_, signedInScript := catalog.App.CallPhysicalAPI(t, catalog.Owner, apptest.PhysicalScriptRequest())
			assert.Equal(t, publicScript, signedInScript)
			assert.NotContains(t, string(publicScript), catalog.App.URL)
			assert.NotContains(t, string(publicScript), token.Token)
		},
	)
}
