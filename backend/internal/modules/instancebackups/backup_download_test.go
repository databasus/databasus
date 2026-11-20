package instancebackups_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // Each scenario moves its own application's clock.
func TestDownloadInstanceBackup(t *testing.T) {
	t.Parallel()

	t.Run("streams the stored encrypted file once without caching", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)
		downloadToken := owner.createDownloadToken(t, backup.Id, getCopy(t, backup, storageID).Id)
		storedContent, err := os.ReadFile(
			owner.App.GetLocalStorageFilePath(owner.getStoredFileName(t, backup, storageID)),
		)
		require.NoError(t, err)

		firstDownload := downloadBackup(t, owner, downloadToken.Token)
		secondDownload := downloadBackup(t, owner, downloadToken.Token)

		require.Equal(t, http.StatusOK, firstDownload.StatusCode(), string(firstDownload.Body))
		assert.Equal(t, storedContent, firstDownload.Body)
		assert.Equal(t, "no-store", firstDownload.HTTPResponse.Header.Get("Cache-Control"))
		snapshotFileName := "databasus-backup-" + backup.SnapshotAt.UTC().Format("2006-01-02-15-04-05") + ".dbbackup"
		assert.Equal(t, snapshotFileName, downloadToken.FileName)
		assert.Contains(t, firstDownload.HTTPResponse.Header.Get("Content-Disposition"), snapshotFileName)
		requireProblemCode(t, secondDownload.StatusCode(), secondDownload.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 401 after the token expired", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)
		downloadToken := owner.createDownloadToken(t, backup.Id, getCopy(t, backup, storageID).Id)

		owner.App.Clock.Advance(5*time.Minute + time.Second)
		download := downloadBackup(t, owner, downloadToken.Token)

		requireProblemCode(t, download.StatusCode(), download.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 409 storage_unavailable when the storage lost the file", func(t *testing.T) {
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)
		downloadToken := owner.createDownloadToken(t, backup.Id, getCopy(t, backup, storageID).Id)
		require.NoError(t, os.Remove(owner.App.GetLocalStorageFilePath(owner.getStoredFileName(t, backup, storageID))))

		download := downloadBackup(t, owner, downloadToken.Token)

		requireProblemCode(t, download.StatusCode(), download.Body, http.StatusConflict,
			apiclient.ProblemCodeStorageUnavailable)
	})
}

func TestCreateInstanceBackupDownloadToken(t *testing.T) {
	t.Parallel()

	t.Run("returns 404 for a copy of another backup", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)

		response, err := owner.RootAdmin.Client.CreateInstanceBackupDownloadTokenWithResponse(t.Context(), uuid.New(),
			apiclient.InstanceBackupDownloadTokenCreate{CopyId: getCopy(t, backup, storageID).Id})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 404 for a copy that was not stored", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		owner.App.RunJobPass(t, productionJob)
		backup := owner.getLatestBackup(t)

		response, err := owner.RootAdmin.Client.CreateInstanceBackupDownloadTokenWithResponse(t.Context(), backup.Id,
			apiclient.InstanceBackupDownloadTokenCreate{CopyId: getCopy(t, backup, storageID).Id})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 403 to an instance admin who is not root admin", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		storageID := owner.createLocalStorage(t, "Archive")
		owner.selectDestinations(t, storageID)
		backup := owner.backUp(t)

		response, err := owner.OtherAdmin.Client.CreateInstanceBackupDownloadTokenWithResponse(t.Context(), backup.Id,
			apiclient.InstanceBackupDownloadTokenCreate{CopyId: getCopy(t, backup, storageID).Id})

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, response.StatusCode(), string(response.Body))
	})
}

func downloadBackup(t *testing.T, owner instanceOwner, token string) *apiclient.DownloadInstanceBackupResponse {
	t.Helper()

	response, err := owner.App.NewClient(t).DownloadInstanceBackupWithResponse(t.Context(),
		&apiclient.DownloadInstanceBackupParams{Token: token})
	require.NoError(t, err)

	return response
}
