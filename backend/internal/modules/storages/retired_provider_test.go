package storages_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestRetiredStorageProvider(t *testing.T) {
	t.Parallel()

	for _, providerType := range []string{"GOOGLE_DRIVE", "RCLONE"} {
		for _, isInstanceAdmin := range []bool{false, true} {
			for _, operation := range []string{"create", "update", "test"} {
				t.Run(
					fmt.Sprintf(
						"returns invalid_request without changes when %s settings are submitted to %s with admin %t",
						providerType,
						operation,
						isInstanceAdmin,
					),
					func(t *testing.T) {
						t.Parallel()
						storageTeam := newTeam(t)
						actor := storageTeam.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
						if isInstanceAdmin {
							actor = storageTeam.admin
							apptest.AddWorkspaceMember(
								t,
								storageTeam.owner,
								storageTeam.workspace.Id,
								actor.Email,
								apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
							)
						}

						var storageRequests atomic.Int64
						destination := httptest.NewServer(
							http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
								storageRequests.Add(1)
								writer.WriteHeader(http.StatusInternalServerError)
							}),
						)
						t.Cleanup(destination.Close)
						savedStorage := storageTeam.createLocalStorage(t, "Archive")
						settings := map[string]string{"type": providerType}
						if providerType == "RCLONE" {
							settings["configContent"] = "[remote]\ntype = webdav\nurl = " + destination.URL + "\nbearer_token = synthetic-secret\n"
							settings["remotePath"] = "/backups"
						} else {
							settings["clientId"] = "synthetic-client"
							settings["clientSecret"] = "synthetic-secret"
							settings["tokenJson"] = `{"refresh_token":"synthetic-token"}`
						}

						requestBody := map[string]any{
							"settings":    settings,
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
							response, requestErr := actor.Client.CreateStorageWithBodyWithResponse(
								t.Context(), "application/json", bytes.NewReader(encodedBody),
							)
							require.NoError(t, requestErr)
							responseStatus, responseBody = response.StatusCode(), response.Body
						case "update":
							response, requestErr := actor.Client.UpdateStorageWithBodyWithResponse(
								t.Context(), savedStorage.Id, "application/json", bytes.NewReader(encodedBody),
							)
							require.NoError(t, requestErr)
							responseStatus, responseBody = response.StatusCode(), response.Body
						case "test":
							response, requestErr := actor.Client.TestStorageSettingsWithBodyWithResponse(
								t.Context(), "application/json", bytes.NewReader(encodedBody),
							)
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

						assert.Zero(t, storageRequests.Load())
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
	}
}
