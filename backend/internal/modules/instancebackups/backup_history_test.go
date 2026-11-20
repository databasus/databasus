package instancebackups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // Each scenario moves its own application's clock between backups.
func TestListInstanceBackups(t *testing.T) {
	t.Parallel()

	t.Run("pages the history newest snapshot first", func(t *testing.T) {
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))
		oldestBackup := owner.backUp(t)
		owner.App.Clock.Advance(time.Hour)
		middleBackup := owner.backUp(t)
		owner.App.Clock.Advance(time.Hour)
		newestBackup := owner.backUp(t)
		limit := 2

		firstPage, err := owner.RootAdmin.Client.ListInstanceBackupsWithResponse(t.Context(),
			&apiclient.ListInstanceBackupsParams{Limit: &limit, After: nil})
		require.NoError(t, err)
		require.NotNil(t, firstPage.JSON200, string(firstPage.Body))
		require.NotNil(t, firstPage.JSON200.NextCursor)
		secondPage, err := owner.RootAdmin.Client.ListInstanceBackupsWithResponse(t.Context(),
			&apiclient.ListInstanceBackupsParams{Limit: &limit, After: firstPage.JSON200.NextCursor})
		require.NoError(t, err)
		require.NotNil(t, secondPage.JSON200, string(secondPage.Body))

		assert.Equal(t, []uuid.UUID{newestBackup.Id, middleBackup.Id}, getBackupIDs(firstPage.JSON200.Items))
		assert.Equal(t, []uuid.UUID{oldestBackup.Id}, getBackupIDs(secondPage.JSON200.Items))
		assert.Nil(t, secondPage.JSON200.NextCursor)
	})

	t.Run("returns 400 for a cursor it did not issue", func(t *testing.T) {
		owner := newInstanceOwner(t)
		after := "not-a-cursor"

		response, err := owner.RootAdmin.Client.ListInstanceBackupsWithResponse(t.Context(),
			&apiclient.ListInstanceBackupsParams{Limit: nil, After: &after})

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		owner := newInstanceOwner(t)

		response, err := owner.App.NewClient(t).ListInstanceBackupsWithResponse(t.Context(), nil)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode(), string(response.Body))
	})
}

func TestGetInstanceBackup(t *testing.T) {
	t.Parallel()

	t.Run("returns 404 for an unknown backup", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.RootAdmin.Client.GetInstanceBackupWithResponse(t.Context(), uuid.New())

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 403 to an instance admin who is not root admin", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		owner.selectDestinations(t, owner.createLocalStorage(t, "Archive"))
		backup := owner.backUp(t)

		response, err := owner.OtherAdmin.Client.GetInstanceBackupWithResponse(t.Context(), backup.Id)

		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, response.StatusCode(), string(response.Body))
	})
}

func getBackupIDs(backups []apiclient.InstanceBackup) []uuid.UUID {
	backupIDs := make([]uuid.UUID, 0, len(backups))
	for _, backup := range backups {
		backupIDs = append(backupIDs, backup.Id)
	}

	return backupIDs
}
