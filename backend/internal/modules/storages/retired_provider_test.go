package storages_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestRetiredStorageProvider(t *testing.T) {
	t.Parallel()

	for _, operation := range []string{"create", "update", "test"} {
		t.Run(
			"returns invalid_request without changes when retired settings are submitted to "+operation,
			func(t *testing.T) {
				t.Parallel()
				storageTeam := newTeam(t)
				savedStorage := storageTeam.createLocalStorage(t, "Archive")
				requestBody := map[string]any{
					"settings": map[string]string{
						"type": "GOOGLE_DRIVE", "clientId": "synthetic-client",
						"clientSecret": "synthetic-secret", "tokenJson": `{"refresh_token":"synthetic-token"}`,
					},
					"workspaceId": storageTeam.workspace.Id,
					"name":        "Changed",
				}

				if operation == "test" {
					delete(requestBody, "name")
				}

				if operation == "update" {
					delete(requestBody, "workspaceId")
				}

				encodedBody, err := json.Marshal(requestBody)
				require.NoError(t, err)

				var responseStatus int
				var responseBody []byte

				switch operation {
				case "create":
					response, requestErr := storageTeam.owner.Client.CreateStorageWithBodyWithResponse(
						t.Context(), "application/json", bytes.NewReader(encodedBody))
					require.NoError(t, requestErr)
					responseStatus, responseBody = response.StatusCode(), response.Body
				case "update":
					response, requestErr := storageTeam.owner.Client.UpdateStorageWithBodyWithResponse(
						t.Context(), savedStorage.Id, "application/json", bytes.NewReader(encodedBody))
					require.NoError(t, requestErr)
					responseStatus, responseBody = response.StatusCode(), response.Body
				case "test":
					response, requestErr := storageTeam.owner.Client.TestStorageSettingsWithBodyWithResponse(
						t.Context(), "application/json", bytes.NewReader(encodedBody))
					require.NoError(t, requestErr)
					responseStatus, responseBody = response.StatusCode(), response.Body
				}

				requireProblem(
					t,
					responseStatus,
					responseBody,
					http.StatusBadRequest,
					apiclient.ProblemCodeInvalidRequest,
				)

				var rejection apiclient.Problem
				require.NoError(t, json.Unmarshal(responseBody, &rejection))
				require.NotNil(t, rejection.Errors)
				assert.Equal(t, []apiclient.FieldError{
					{Field: "request", Detail: "must match the API contract"},
				}, *rejection.Errors)

				assert.NotContains(t, string(responseBody), "synthetic-secret")
				assert.NotContains(t, string(responseBody), "synthetic-token")
				assert.Equal(t, savedStorage, getStorage(t, storageTeam.owner, savedStorage.Id))
				assert.Equal(
					t,
					[]apiclient.Storage{savedStorage},
					listStorages(t, storageTeam.owner, storageTeam.workspace.Id),
				)
			},
		)
	}
}
