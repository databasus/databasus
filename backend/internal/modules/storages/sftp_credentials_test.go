package storages_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSftpCredentials(t *testing.T) {
	t.Parallel()

	for _, methods := range [][]apiclient.SftpAuthMethod{
		{apiclient.SftpAuthMethodPASSWORD},
		{apiclient.SftpAuthMethodPRIVATEKEY},
		{apiclient.SftpAuthMethodPASSWORD, apiclient.SftpAuthMethodPRIVATEKEY},
	} {
		t.Run(
			"preserves encrypted credentials after unrelated edits with "+strings.Join(
				getSftpMethodNames(methods),
				" and ",
			)+" credentials",
			func(t *testing.T) {
				t.Parallel()

				storageTeam := newTeam(t)
				settings := newSftpSettingsForMethods(methods)

				createdStorage := createStorage(t, storageTeam.owner, apiclient.StorageCreate{
					WorkspaceId: storageTeam.workspace.Id, Name: "Archive", Settings: newSftpSettings(t, settings),
				})

				requireSftpMethods(t, createdStorage, methods)
				encryptedCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)
				assert.NotContains(t, encryptedCredentials.password, "original-password")
				assert.NotContains(t, encryptedCredentials.privateKey, sftpPrivateKey)

				settings.Password, settings.PrivateKey = nil, nil
				settings.Path = new("/renamed")

				updatedStorage := updateStorage(t, storageTeam.owner, createdStorage.Id, apiclient.StorageUpdate{
					Name: "Renamed", Settings: newSftpSettings(t, settings),
				})

				require.Equal(t, http.StatusOK, updatedStorage.StatusCode(), string(updatedStorage.Body))
				requireSftpMethods(t, *updatedStorage.JSON200, methods)
				requireSftpMethods(t, getStorage(t, storageTeam.owner, createdStorage.Id), methods)
				requireSftpMethods(t, listStorages(t, storageTeam.owner, storageTeam.workspace.Id)[0], methods)
				assert.Equal(t, encryptedCredentials, getStoredSftpCredentials(t, storageTeam, createdStorage.Id))
			},
		)
	}

	t.Run("replaces one credential and clears excluded credentials after explicit switches", func(t *testing.T) {
		t.Parallel()

		storageTeam := newTeam(t)
		settings := newSftpSettingsForMethods(
			[]apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD, apiclient.SftpAuthMethodPRIVATEKEY},
		)

		createdStorage := createStorage(
			t,
			storageTeam.owner,
			apiclient.StorageCreate{
				WorkspaceId: storageTeam.workspace.Id,
				Name:        "Archive",
				Settings:    newSftpSettings(t, settings),
			},
		)
		encryptedCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)
		settings.Password, settings.PrivateKey = new("replacement-password"), nil

		replacement := updateStorage(
			t,
			storageTeam.owner,
			createdStorage.Id,
			apiclient.StorageUpdate{Name: "Archive", Settings: newSftpSettings(t, settings)},
		)

		require.Equal(t, http.StatusOK, replacement.StatusCode(), string(replacement.Body))
		replacedCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)
		assert.Equal(t, encryptedCredentials.privateKey, replacedCredentials.privateKey)
		assert.Equal(t, "replacement-password", storageTeam.app.DecryptSecret(t, replacedCredentials.password))

		settings.AuthMethods, settings.Password = []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPRIVATEKEY}, nil

		keySwitch := updateStorage(
			t,
			storageTeam.owner,
			createdStorage.Id,
			apiclient.StorageUpdate{Name: "Key archive", Settings: newSftpSettings(t, settings)},
		)

		require.Equal(t, http.StatusOK, keySwitch.StatusCode(), string(keySwitch.Body))
		assert.Empty(t, getStoredSftpCredentials(t, storageTeam, createdStorage.Id).password)
		assert.Equal(
			t,
			encryptedCredentials.privateKey,
			getStoredSftpCredentials(t, storageTeam, createdStorage.Id).privateKey,
		)

		settings.AuthMethods = []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD}

		refusedSwitch := updateStorage(
			t,
			storageTeam.owner,
			createdStorage.Id,
			apiclient.StorageUpdate{Name: "Refused", Settings: newSftpSettings(t, settings)},
		)

		requireProblem(
			t,
			refusedSwitch.StatusCode(),
			refusedSwitch.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
		assert.Equal(t, "Key archive", getStorage(t, storageTeam.owner, createdStorage.Id).Name)
		assert.Equal(
			t,
			encryptedCredentials.privateKey,
			getStoredSftpCredentials(t, storageTeam, createdStorage.Id).privateKey,
		)

		settings.Password = new("new-password")

		passwordSwitch := updateStorage(
			t,
			storageTeam.owner,
			createdStorage.Id,
			apiclient.StorageUpdate{Name: "Password archive", Settings: newSftpSettings(t, settings)},
		)

		require.Equal(t, http.StatusOK, passwordSwitch.StatusCode(), string(passwordSwitch.Body))
		assert.Empty(t, getStoredSftpCredentials(t, storageTeam, createdStorage.Id).privateKey)
		assert.Equal(
			t,
			"new-password",
			storageTeam.app.DecryptSecret(t, getStoredSftpCredentials(t, storageTeam, createdStorage.Id).password),
		)
		requireSftpMethods(t, getStorage(t, storageTeam.owner, createdStorage.Id), settings.AuthMethods)
	})
}

func TestSftpInvalidCredentials(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		settings map[string]any
	}{
		{
			"missing methods",
			map[string]any{"password": "request-only-password", "privateKey": "request-only-private-key"},
		},
		{"empty methods", map[string]any{"authMethods": []string{}, "password": "secret"}},
		{"duplicate methods", map[string]any{"authMethods": []string{"PASSWORD", "PASSWORD"}, "password": "secret"}},
		{"unsupported method", map[string]any{"authMethods": []string{"TOKEN"}, "password": "secret"}},
		{"empty password", map[string]any{"authMethods": []string{"PASSWORD"}, "password": ""}},
		{"empty private key", map[string]any{"authMethods": []string{"PRIVATE_KEY"}, "privateKey": ""}},
		{
			"excluded private key",
			map[string]any{"authMethods": []string{"PASSWORD"}, "password": "secret", "privateKey": sftpPrivateKey},
		},
		{
			"excluded password",
			map[string]any{"authMethods": []string{"PRIVATE_KEY"}, "password": "secret", "privateKey": sftpPrivateKey},
		},
		{"missing selected key", map[string]any{"authMethods": []string{"PRIVATE_KEY"}}},
	}

	for _, testCase := range testCases {
		t.Run("rejects requests before connecting with "+testCase.name, func(t *testing.T) {
			t.Parallel()

			storageTeam := newTeam(t)
			listener := startSilentListener(t)
			savedSettings := newSftpSettingsForMethods([]apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD})
			createdStorage := createStorage(
				t,
				storageTeam.owner,
				apiclient.StorageCreate{
					WorkspaceId: storageTeam.workspace.Id,
					Name:        "Archive",
					Settings:    newSftpSettings(t, savedSettings),
				},
			)
			encryptedCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)
			settings := map[string]any{
				"type":                "SFTP",
				"host":                listener.host,
				"port":                listener.port,
				"username":            "backup",
				"isSkipHostKeyVerify": true,
			}
			maps.Copy(settings, testCase.settings)

			for _, operation := range []string{"create", "update", "test"} {
				body := map[string]any{"settings": settings, "name": "Changed"}
				if operation != "update" {
					body["workspaceId"] = storageTeam.workspace.Id
				}

				if operation == "test" {
					delete(body, "name")
					body["storageId"] = createdStorage.Id
				}

				encodedBody, err := json.Marshal(body)
				require.NoError(t, err)

				switch operation {
				case "create":
					response, err := storageTeam.owner.Client.CreateStorageWithBodyWithResponse(
						t.Context(),
						"application/json",
						bytes.NewReader(encodedBody),
					)
					require.NoError(t, err)

					requireProblem(
						t,
						response.StatusCode(),
						response.Body,
						http.StatusBadRequest,
						apiclient.ProblemCodeInvalidRequest,
					)
					requireRejectedSftpSecretsHidden(t, response.Body, settings)

				case "update":
					response, err := storageTeam.owner.Client.UpdateStorageWithBodyWithResponse(
						t.Context(),
						createdStorage.Id,
						"application/json",
						bytes.NewReader(encodedBody),
					)
					require.NoError(t, err)

					requireProblem(
						t,
						response.StatusCode(),
						response.Body,
						http.StatusBadRequest,
						apiclient.ProblemCodeInvalidRequest,
					)
					requireRejectedSftpSecretsHidden(t, response.Body, settings)

				case "test":
					response, err := storageTeam.owner.Client.TestStorageSettingsWithBodyWithResponse(
						t.Context(),
						"application/json",
						bytes.NewReader(encodedBody),
					)
					require.NoError(t, err)

					requireProblem(
						t,
						response.StatusCode(),
						response.Body,
						http.StatusBadRequest,
						apiclient.ProblemCodeInvalidRequest,
					)
					requireRejectedSftpSecretsHidden(t, response.Body, settings)
				}
			}

			assert.False(t, listener.isConnectionPending(t))
			assert.Equal(t, encryptedCredentials, getStoredSftpCredentials(t, storageTeam, createdStorage.Id))
			assert.Equal(t, "Archive", getStorage(t, storageTeam.owner, createdStorage.Id).Name)
			assert.Len(t, listStorages(t, storageTeam.owner, storageTeam.workspace.Id), 1)
		})
	}
}

type sftpCredentials struct {
	password   string
	privateKey string
}

func getStoredSftpCredentials(t *testing.T, storageTeam team, storageID uuid.UUID) sftpCredentials {
	t.Helper()

	var rows []struct {
		Password   string
		PrivateKey string
	}
	storageTeam.app.QuerySQL(
		t,
		&rows,
		"SELECT COALESCE(password, '') AS password, COALESCE(private_key, '') AS private_key FROM storages_sftp WHERE storage_id = ?",
		storageID,
	)
	require.Len(t, rows, 1)

	return sftpCredentials{password: rows[0].Password, privateKey: rows[0].PrivateKey}
}

func newSftpSettingsForMethods(methods []apiclient.SftpAuthMethod) apiclient.SftpStorageSettings {
	settings := apiclient.SftpStorageSettings{
		Type:                apiclient.SftpStorageSettingsTypeSFTP,
		Host:                "sftp.example.com",
		Port:                22,
		Username:            "backup",
		AuthMethods:         methods,
		IsSkipHostKeyVerify: true,
		Password:            nil,
		PrivateKey:          nil,
		Path:                nil,
	}

	for _, method := range methods {
		if method == apiclient.SftpAuthMethodPASSWORD {
			settings.Password = new("original-password")
		}

		if method == apiclient.SftpAuthMethodPRIVATEKEY {
			settings.PrivateKey = new(sftpPrivateKey)
		}
	}

	return settings
}

func requireSftpMethods(t *testing.T, storage apiclient.Storage, methods []apiclient.SftpAuthMethod) {
	t.Helper()

	settings, err := storage.Settings.AsSftpStorageSettings()
	require.NoError(t, err)

	assert.Equal(t, methods, settings.AuthMethods)
	assert.Nil(t, settings.Password)
	assert.Nil(t, settings.PrivateKey)
}

func TestSftpCredentialAccess(t *testing.T) {
	t.Parallel()

	t.Run("preserves credentials and connects nowhere when the actor cannot manage the storage", func(t *testing.T) {
		t.Parallel()

		storageTeam := newTeam(t)
		viewer := storageTeam.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		stranger := storageTeam.stranger(t)
		settings := newSftpSettingsForMethods([]apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD})

		createdStorage := createStorage(
			t,
			storageTeam.owner,
			apiclient.StorageCreate{
				WorkspaceId: storageTeam.workspace.Id,
				Name:        "Archive",
				Settings:    newSftpSettings(t, settings),
			},
		)
		encryptedCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)
		listener := startSilentListener(t)
		settings.Host, settings.Port = listener.host, listener.port

		for _, actor := range []struct {
			client *apiclient.ClientWithResponses
			status int
			code   apiclient.ProblemCode
		}{
			{storageTeam.app.NewClient(t), http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated},
			{viewer.Client, http.StatusForbidden, apiclient.ProblemCodePermissionDenied},
			{stranger.Client, http.StatusNotFound, apiclient.ProblemCodeNotFound},
		} {
			creation, err := actor.client.CreateStorageWithResponse(
				t.Context(),
				apiclient.StorageCreate{
					WorkspaceId: storageTeam.workspace.Id,
					Name:        "Refused",
					Settings:    newSftpSettings(t, settings),
				},
			)
			require.NoError(t, err)

			requireProblem(t, creation.StatusCode(), creation.Body, actor.status, actor.code)

			update, err := actor.client.UpdateStorageWithResponse(
				t.Context(),
				createdStorage.Id,
				apiclient.StorageUpdate{Name: "Refused", Settings: newSftpSettings(t, settings)},
			)
			require.NoError(t, err)

			requireProblem(t, update.StatusCode(), update.Body, actor.status, actor.code)

			connection, err := actor.client.TestStorageSettingsWithResponse(
				t.Context(),
				apiclient.StorageSettingsTest{
					WorkspaceId: storageTeam.workspace.Id,
					StorageId:   &createdStorage.Id,
					Settings:    newSftpSettings(t, settings),
				},
			)
			require.NoError(t, err)

			requireProblem(t, connection.StatusCode(), connection.Body, actor.status, actor.code)
		}

		strangerWorkspace := apptest.CreateWorkspace(t, stranger, "Other workspace")

		foreignTest, err := stranger.Client.TestStorageSettingsWithResponse(
			t.Context(),
			apiclient.StorageSettingsTest{
				WorkspaceId: strangerWorkspace.Id,
				StorageId:   &createdStorage.Id,
				Settings:    newSftpSettings(t, settings),
			},
		)
		require.NoError(t, err)

		requireProblem(
			t,
			foreignTest.StatusCode(),
			foreignTest.Body,
			http.StatusNotFound,
			apiclient.ProblemCodeNotFound,
		)
		assert.False(t, listener.isConnectionPending(t))
		assert.Equal(t, encryptedCredentials, getStoredSftpCredentials(t, storageTeam, createdStorage.Id))
		assert.Equal(t, "Archive", getStorage(t, storageTeam.owner, createdStorage.Id).Name)
		assert.Len(t, listStorages(t, storageTeam.owner, storageTeam.workspace.Id), 1)
	})
}

func getSftpMethodNames(methods []apiclient.SftpAuthMethod) []string {
	names := make([]string, len(methods))
	for index, method := range methods {
		names[index] = string(method)
	}

	return names
}

func requireRejectedSftpSecretsHidden(t *testing.T, body []byte, settings map[string]any) {
	t.Helper()

	var responseBody map[string]any
	require.NoError(t, json.Unmarshal(body, &responseBody))

	for _, field := range []string{"password", "privateKey"} {
		credential, hasCredential := settings[field].(string)

		if hasCredential && credential != "" {
			assert.NotContains(t, fmt.Sprint(responseBody), credential)
		}
	}
}
