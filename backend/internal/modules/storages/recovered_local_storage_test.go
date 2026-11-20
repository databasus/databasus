package storages_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/storages"
)

func TestLocalStorageAfterRecovery(t *testing.T) {
	t.Parallel()

	t.Run("writes a file to a local storage right after a recovery without a connection test", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithPublishedRecovery())
		localStorage := tm.createLocalStorage(t, "Recovered")
		tm.app.NormalizeRecoveredStorages(t)

		err := tm.app.WriteStorageFile(t.Context(),
			storages.StoredFileReference{StorageID: localStorage.Id, FileName: "new.dump"},
			strings.NewReader("new backup"))

		require.NoError(t, err)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, "new.dump"))
	})

	t.Run("reports an editor's test of a local storage right after a recovery and changes nothing",
		func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t, apptest.WithPublishedRecovery())
			editor := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			localStorage := tm.createLocalStorage(t, "Recovered")
			tm.app.NormalizeRecoveredStorages(t)
			rowsBefore := getStoredLocalStorageRows(t, tm.app, localStorage.Id)

			testResult := testSavedStorage(t, editor, localStorage.Id)

			assert.True(t, testResult.IsConnected)
			assert.Equal(t, rowsBefore, getStoredLocalStorageRows(t, tm.app, localStorage.Id))
		})

	t.Run("cleans up an unpublished file of a local storage right after a recovery", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithPublishedRecovery())
		localStorage := tm.createLocalStorage(t, "Recovered")
		tm.app.NormalizeRecoveredStorages(t)
		tm.app.WriteUnpublishedFile(t, localStorage.Id, "unpublished.dump")
		tm.app.Clock.Advance(time.Hour)

		tm.app.RunJobPass(t, "storage_file_deletion")

		assert.False(t, tm.app.IsLocalStorageFilePresent(t, "unpublished.dump"))
	})
}

// getStoredLocalStorageRows reads every column of a local storage and of its provider row as JSON.
func getStoredLocalStorageRows(t *testing.T, app *apptest.App, storageID uuid.UUID) []string {
	t.Helper()

	var storedRows []string

	for _, query := range []string{
		"SELECT row_to_json(s)::text FROM storages s WHERE id = ?",
		"SELECT row_to_json(p)::text FROM storages_local p WHERE storage_id = ?",
	} {
		var rows []string
		app.QuerySQL(t, &rows, query, storageID)
		storedRows = append(storedRows, rows...)
	}

	require.Len(t, storedRows, 2, fmt.Sprint(storedRows))

	return storedRows
}
