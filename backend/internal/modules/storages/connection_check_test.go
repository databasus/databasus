package storages_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestTestStorage(t *testing.T) {
	t.Parallel()

	t.Run("reports a local storage as connected when a viewer tests it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		localStorage := tm.createLocalStorage(t, "Local")

		testResult := testSavedStorage(t, viewer, localStorage.Id)

		assert.True(t, testResult.IsConnected)
		assert.Nil(t, testResult.Error)
	})

	t.Run("reports an S3 storage as not connected with the missing bucket when the bucket does not exist",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			bucket := startMinioBucket(t)
			missingBucketName := "missing-" + uuid.NewString()[:8]
			s3Storage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Offsite",
				Settings: newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:    missingBucketName,
					S3Region:    new(containers.MinioRegion),
					S3Endpoint:  new(bucket.endpointURL),
					S3AccessKey: new(containers.MinioRootUser),
					S3SecretKey: new(containers.MinioRootPassword),
				}),
			})

			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

			testResult := testSavedStorage(t, member, s3Storage.Id)

			assert.False(t, testResult.IsConnected)
			require.NotNil(t, testResult.Error)
			assert.Contains(t, *testResult.Error, missingBucketName)
		})

	t.Run("returns 404 not_found when a stranger tests another team's storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.stranger(t)
		localStorage := tm.createLocalStorage(t, "Local")

		response, err := stranger.Client.TestStorageWithResponse(t.Context(), localStorage.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Local")

		response, err := tm.app.NewClient(t).TestStorageWithResponse(t.Context(), localStorage.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("changes no stored column when an S3 and then a local storage are tested", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		bucket := startMinioBucket(t)
		s3StorageID := insertStoredS3Storage(t, tm, storedS3Storage{
			bucket:             bucket,
			encryptedAccessKey: tm.app.EncryptSecret(t, containers.MinioRootUser),
			encryptedSecretKey: tm.app.EncryptSecret(t, containers.MinioRootPassword),
		})
		localStorage := tm.createLocalStorage(t, "Local")
		rowsBefore := getStoredStorageRows(t, tm.app, s3StorageID, localStorage.Id)

		s3TestResult := testSavedStorage(t, tm.owner, s3StorageID)
		localTestResult := testSavedStorage(t, tm.owner, localStorage.Id)

		require.True(t, s3TestResult.IsConnected, "the S3 test must reach the bucket: %v", s3TestResult.Error)
		require.True(t, localTestResult.IsConnected)
		assert.Equal(t, rowsBefore, getStoredStorageRows(t, tm.app, s3StorageID, localStorage.Id))
	})
}

func testSavedStorage(t *testing.T, session apptest.Session, storageID uuid.UUID) apiclient.StorageTestResult {
	t.Helper()

	response, err := session.Client.TestStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

// getStoredStorageRows reads every column of the storages and of their provider rows as JSON, so a
// test can show that nothing a test touched was written.
func getStoredStorageRows(t *testing.T, app *apptest.App, s3StorageID, localStorageID uuid.UUID) []string {
	t.Helper()

	storageIDs := []uuid.UUID{s3StorageID, localStorageID}
	storedRows := make([]string, 0, 4)

	for _, query := range []string{
		"SELECT row_to_json(s)::text FROM storages s WHERE id IN ? ORDER BY id",
		"SELECT row_to_json(p)::text FROM storages_s3 p WHERE storage_id IN ? ORDER BY storage_id",
		"SELECT row_to_json(p)::text FROM storages_local p WHERE storage_id IN ? ORDER BY storage_id",
	} {
		var rows []string
		app.QuerySQL(t, &rows, query, storageIDs)
		storedRows = append(storedRows, rows...)
	}

	require.Len(t, storedRows, 4, fmt.Sprint(storedRows))

	return storedRows
}
