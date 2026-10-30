package storages_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestUpdateStorage(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 invalid_request and keeps the local storage when S3 settings are sent", func(t *testing.T) {
		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Local")

		response := updateStorage(t, tm.owner, localStorage.Id, apiclient.StorageUpdate{
			Name: "Now S3",
			Settings: newS3Settings(t, apiclient.S3StorageSettings{
				S3Bucket:    "backups",
				S3AccessKey: new("access"),
				S3SecretKey: new("secret"),
			}),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		storedStorage := getStorage(t, tm.owner, localStorage.Id)
		assert.Equal(t, "Local", storedStorage.Name)
		_, err := storedStorage.Settings.AsLocalStorageSettings()
		require.NoError(t, err)
		storageType, err := storedStorage.Settings.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "LOCAL", storageType)
	})

	t.Run("returns 400 invalid_request and keeps the port when an FTP storage gets port 70000", func(t *testing.T) {
		tm := newTeam(t)
		ftpStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "FTP",
			Settings: newFtpSettings(t, apiclient.FtpStorageSettings{
				Host:     "ftp.example.com",
				Port:     21,
				Username: "backup",
				Password: new("password"),
			}),
		})

		response := updateStorage(t, tm.owner, ftpStorage.Id, apiclient.StorageUpdate{
			Name: "FTP",
			Settings: newFtpSettings(t, apiclient.FtpStorageSettings{
				Host:     "ftp.example.com",
				Port:     70000,
				Username: "backup",
			}),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		storedFtp, err := getStorage(t, tm.owner, ftpStorage.Id).Settings.AsFtpStorageSettings()
		require.NoError(t, err)
		assert.Equal(t, 21, storedFtp.Port)
	})

	t.Run("returns 400 invalid_request and keeps the name when the name is empty", func(t *testing.T) {
		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Local")

		response := updateStorage(t, tm.owner, localStorage.Id, apiclient.StorageUpdate{
			Name:     "",
			Settings: newLocalSettings(t),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.Equal(t, "Local", getStorage(t, tm.owner, localStorage.Id).Name)
	})
}

func TestCreateInvalidStorage(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 invalid_request and creates nothing when the S3 storage has no bucket", func(t *testing.T) {
		tm := newTeam(t)

		response, err := tm.owner.Client.CreateStorageWithResponse(t.Context(), apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Offsite",
			Settings: newS3Settings(t, apiclient.S3StorageSettings{
				S3Bucket:    "",
				S3AccessKey: new("access"),
				S3SecretKey: new("secret"),
			}),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		tm := newTeam(t)

		response, err := tm.app.NewClient(t).CreateStorageWithResponse(t.Context(), apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Local",
			Settings:    newLocalSettings(t),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})
}
