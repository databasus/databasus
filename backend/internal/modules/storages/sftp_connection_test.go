package storages_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestSftpSelectedConnection(t *testing.T) {
	t.Parallel()

	t.Run("excludes a valid saved password when a key-only test is refused", func(t *testing.T) {
		t.Parallel()

		privateKey, _ := newSftpKey(t)
		server := containers.StartSftp(t)
		storageTeam := newTeam(t)
		settings := newSftpSettingsForMethods([]apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD})
		settings.Host, settings.Port, settings.Username = server.Host, server.Port, containers.SftpUsername
		settings.Password, settings.Path = new(containers.SftpPassword), new(containers.SftpUploadDir)

		createdStorage := createStorage(
			t,
			storageTeam.owner,
			apiclient.StorageCreate{
				WorkspaceId: storageTeam.workspace.Id,
				Name:        "Archive",
				Settings:    newSftpSettings(t, settings),
			},
		)
		storedCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)
		storageTeam.app.WriteUnpublishedFile(t, createdStorage.Id, "before-switch")
		settings.AuthMethods, settings.Password, settings.PrivateKey = []apiclient.SftpAuthMethod{
			apiclient.SftpAuthMethodPRIVATEKEY,
		}, nil, new(
			privateKey,
		)

		connection := testUnsavedSettings(
			t,
			storageTeam.owner,
			apiclient.StorageSettingsTest{
				WorkspaceId: storageTeam.workspace.Id,
				StorageId:   &createdStorage.Id,
				Settings:    newSftpSettings(t, settings),
			},
		)

		assert.False(t, connection.IsConnected)
		assert.Equal(t, storedCredentials, getStoredSftpCredentials(t, storageTeam, createdStorage.Id))
		requireSftpMethods(
			t,
			getStorage(t, storageTeam.owner, createdStorage.Id),
			[]apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD},
		)

		settings.AuthMethods, settings.PrivateKey = []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD}, nil

		retained := testUnsavedSettings(
			t,
			storageTeam.owner,
			apiclient.StorageSettingsTest{
				WorkspaceId: storageTeam.workspace.Id,
				StorageId:   &createdStorage.Id,
				Settings:    newSftpSettings(t, settings),
			},
		)

		assert.True(t, retained.IsConnected)
		assert.Equal(t, storedCredentials, getStoredSftpCredentials(t, storageTeam, createdStorage.Id))
	})

	t.Run("uses only the saved key after switching to a key-only server", func(t *testing.T) {
		t.Parallel()

		privateKey, publicKey := newSftpKey(t)
		passwordServer := containers.StartSftp(t)
		keyServer := containers.StartSftpWithCredentials(
			t,
			containers.SftpCredentials{Password: "", PublicKey: publicKey},
		)
		storageTeam := newTeam(t)
		settings := newSftpSettingsForMethods([]apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD})
		settings.Host, settings.Port, settings.Username = passwordServer.Host, passwordServer.Port, containers.SftpUsername
		settings.Password, settings.Path = new(containers.SftpPassword), new(containers.SftpUploadDir)

		createdStorage := createStorage(
			t,
			storageTeam.owner,
			apiclient.StorageCreate{
				WorkspaceId: storageTeam.workspace.Id,
				Name:        "Archive",
				Settings:    newSftpSettings(t, settings),
			},
		)
		storageTeam.app.WriteUnpublishedFile(t, createdStorage.Id, "password-file")
		storedCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)

		settings.Host, settings.Port = keyServer.Host, keyServer.Port
		settings.AuthMethods, settings.Password, settings.PrivateKey = []apiclient.SftpAuthMethod{
			apiclient.SftpAuthMethodPRIVATEKEY,
		}, nil, new(
			privateKey,
		)

		connection := testUnsavedSettings(
			t,
			storageTeam.owner,
			apiclient.StorageSettingsTest{
				WorkspaceId: storageTeam.workspace.Id,
				StorageId:   &createdStorage.Id,
				Settings:    newSftpSettings(t, settings),
			},
		)

		require.True(t, connection.IsConnected, "%v", connection.Error)
		assert.Equal(t, storedCredentials, getStoredSftpCredentials(t, storageTeam, createdStorage.Id))

		switched := updateStorage(
			t,
			storageTeam.owner,
			createdStorage.Id,
			apiclient.StorageUpdate{Name: "Key archive", Settings: newSftpSettings(t, settings)},
		)

		require.Equal(t, http.StatusOK, switched.StatusCode(), string(switched.Body))

		storageTeam.app.WriteUnpublishedFile(t, createdStorage.Id, "key-file")
		settings.PrivateKey = nil

		retained := testUnsavedSettings(
			t,
			storageTeam.owner,
			apiclient.StorageSettingsTest{
				WorkspaceId: storageTeam.workspace.Id,
				StorageId:   &createdStorage.Id,
				Settings:    newSftpSettings(t, settings),
			},
		)

		require.True(t, retained.IsConnected, "%v", retained.Error)
		keyCredentials := getStoredSftpCredentials(t, storageTeam, createdStorage.Id)
		assert.Empty(t, keyCredentials.password)
		assert.Equal(t, privateKey, storageTeam.app.DecryptSecret(t, keyCredentials.privateKey))
		requireSftpMethods(t, getStorage(t, storageTeam.owner, createdStorage.Id), settings.AuthMethods)

		settings.Host, settings.Port = passwordServer.Host, passwordServer.Port
		settings.AuthMethods, settings.Password = []apiclient.SftpAuthMethod{
			apiclient.SftpAuthMethodPASSWORD,
		}, new(
			containers.SftpPassword,
		)

		switchedBack := updateStorage(
			t,
			storageTeam.owner,
			createdStorage.Id,
			apiclient.StorageUpdate{Name: "Password archive", Settings: newSftpSettings(t, settings)},
		)

		require.Equal(t, http.StatusOK, switchedBack.StatusCode(), string(switchedBack.Body))

		storageTeam.app.WriteUnpublishedFile(t, createdStorage.Id, "password-file-after-switch")

		assert.Empty(t, getStoredSftpCredentials(t, storageTeam, createdStorage.Id).privateKey)
	})
}

func newSftpKey(t *testing.T) (string, string) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	privateBlock, err := ssh.MarshalPrivateKey(privateKey, "backup")
	require.NoError(t, err)

	authorizedKey, err := ssh.NewPublicKey(publicKey)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(privateBlock)), string(ssh.MarshalAuthorizedKey(authorizedKey))
}

func TestSftpConnectionIdentity(t *testing.T) {
	t.Parallel()

	t.Run(
		"opens a key-authenticated session after a password session is pooled on the same server",
		func(t *testing.T) {
			t.Parallel()

			privateKey, publicKey := newSftpKey(t)
			server := containers.StartSftpWithCredentials(
				t,
				containers.SftpCredentials{Password: containers.SftpPassword, PublicKey: publicKey},
			)
			storageTeam := newTeam(t)
			settings := newSftpSettingsForMethods(
				[]apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD, apiclient.SftpAuthMethodPRIVATEKEY},
			)
			settings.Host, settings.Port, settings.Username = server.Host, server.Port, containers.SftpUsername
			settings.Password, settings.PrivateKey, settings.Path = new(
				containers.SftpPassword,
			), new(
				privateKey,
			), new(
				containers.SftpUploadDir,
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
			settings.Password, settings.PrivateKey = nil, nil

			retained := testUnsavedSettings(
				t,
				storageTeam.owner,
				apiclient.StorageSettingsTest{
					WorkspaceId: storageTeam.workspace.Id,
					StorageId:   &createdStorage.Id,
					Settings:    newSftpSettings(t, settings),
				},
			)

			require.True(t, retained.IsConnected, "%v", retained.Error)

			savedTest, err := storageTeam.owner.Client.TestStorageWithResponse(t.Context(), createdStorage.Id)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, savedTest.StatusCode(), string(savedTest.Body))
			require.True(t, savedTest.JSON200.IsConnected)
			assert.Equal(t, encryptedCredentials, getStoredSftpCredentials(t, storageTeam, createdStorage.Id))

			storageTeam.app.WriteUnpublishedFile(t, createdStorage.Id, "pooled-password-file")
			settings.AuthMethods = []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPRIVATEKEY}

			switched := updateStorage(
				t,
				storageTeam.owner,
				createdStorage.Id,
				apiclient.StorageUpdate{Name: "Key archive", Settings: newSftpSettings(t, settings)},
			)

			require.Equal(t, http.StatusOK, switched.StatusCode(), string(switched.Body))

			storageTeam.app.WriteUnpublishedFile(t, createdStorage.Id, "new-key-session-file")

			serverLogs, err := server.Container.Logs(t.Context())
			require.NoError(t, err)
			defer func() { _ = serverLogs.Close() }()
			logBytes, err := io.ReadAll(serverLogs)
			require.NoError(t, err)

			assert.Contains(t, string(logBytes), "Accepted password for "+containers.SftpUsername)
			assert.Contains(t, string(logBytes), "Accepted publickey for "+containers.SftpUsername)
		},
	)
}
