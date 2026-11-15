package storages_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/storages"
)

func TestRecoveredLocalStorageValidation(t *testing.T) {
	t.Parallel()

	t.Run("keeps a recovered local storage unavailable after a viewer's successful test", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		localStorage := tm.createLocalStorage(t, "Recovered")
		tm.app.MarkStorageRecovered(t, localStorage.Id)

		testResult := testSavedStorage(t, viewer, localStorage.Id)

		assert.True(t, testResult.IsConnected)
		require.NotNil(t, testResult.RecoveryValidationStatus)
		assert.Equal(
			t,
			apiclient.StorageRecoveryValidationStatusREQUIRESVALIDATION,
			*testResult.RecoveryValidationStatus,
		)
		assert.Equal(t, apiclient.StorageRecoveryValidationStatusREQUIRESVALIDATION,
			getStorage(t, tm.owner, localStorage.Id).RecoveryValidationStatus)
		assertNoRecoveryProbeLeft(t, tm.app)
	})

	t.Run("makes a recovered local storage available after an editor's test, changing only its status",
		func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			localStorage := tm.createLocalStorage(t, "Recovered")
			tm.app.MarkStorageRecovered(t, localStorage.Id)
			existingBackupPath := writeExistingBackupFile(t, tm.app)
			rowsBefore := getRowsWithoutStatus(t, tm.app, localStorage.Id)

			testResult := testSavedStorage(t, tm.owner, localStorage.Id)

			assert.True(t, testResult.IsConnected)
			require.NotNil(t, testResult.RecoveryValidationStatus)
			assert.Equal(t, apiclient.StorageRecoveryValidationStatusAVAILABLE, *testResult.RecoveryValidationStatus)
			assert.Equal(t, apiclient.StorageRecoveryValidationStatusAVAILABLE,
				getStorage(t, tm.owner, localStorage.Id).RecoveryValidationStatus)
			assert.Equal(t, rowsBefore, getRowsWithoutStatus(t, tm.app, localStorage.Id))
			existingContent, err := os.ReadFile(existingBackupPath)
			require.NoError(t, err)
			assert.Equal(t, "existing backup", string(existingContent))
			assertNoRecoveryProbeLeft(t, tm.app)
		})

	t.Run("keeps it unavailable when only the temporary folder is writable", func(t *testing.T) {
		t.Parallel()
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Recovered")
		tm.app.MarkStorageRecovered(t, localStorage.Id)
		existingBackupPath := writeExistingBackupFile(t, tm.app)
		require.NoError(t, os.Chmod(tm.app.GetDataFolder(), 0o500))
		t.Cleanup(func() { _ = os.Chmod(tm.app.GetDataFolder(), 0o755) })

		testResult := testSavedStorage(t, tm.owner, localStorage.Id)

		assert.False(t, testResult.IsConnected)
		require.NotNil(t, testResult.Error)
		assert.Contains(t, *testResult.Error, "backup folder")
		require.NotNil(t, testResult.RecoveryValidationStatus)
		assert.Equal(
			t,
			apiclient.StorageRecoveryValidationStatusREQUIRESVALIDATION,
			*testResult.RecoveryValidationStatus,
		)
		existingContent, err := os.ReadFile(existingBackupPath)
		require.NoError(t, err)
		assert.Equal(t, "existing backup", string(existingContent))
	})

	t.Run("keeps it unavailable when the backup folder is missing", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Recovered")
		tm.app.MarkStorageRecovered(t, localStorage.Id)
		require.NoError(t, os.RemoveAll(tm.app.GetDataFolder()))

		testResult := testSavedStorage(t, tm.owner, localStorage.Id)

		assert.False(t, testResult.IsConnected)
		assert.Equal(t, apiclient.StorageRecoveryValidationStatusREQUIRESVALIDATION,
			getStorage(t, tm.owner, localStorage.Id).RecoveryValidationStatus)
	})

	t.Run("reports an ordinary local storage as available without the recovery probe", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Ordinary")

		testResult := testSavedStorage(t, tm.owner, localStorage.Id)

		assert.True(t, testResult.IsConnected)
		require.NotNil(t, testResult.RecoveryValidationStatus)
		assert.Equal(t, apiclient.StorageRecoveryValidationStatusAVAILABLE, *testResult.RecoveryValidationStatus)
	})
}

func TestRecoveredLocalStorageFiles(t *testing.T) {
	t.Parallel()

	t.Run("refuses writes to a recovered local storage with storage_unavailable", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Recovered")
		tm.app.MarkStorageRecovered(t, localStorage.Id)

		err := tm.app.WriteStorageFile(t.Context(),
			storages.StoredFileReference{StorageID: localStorage.Id, FileName: "new.dump"},
			strings.NewReader("new backup"))

		requireProblemCode(t, err, apierror.CodeStorageUnavailable)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, "new.dump"))
	})

	t.Run("keeps the files of a recovered local storage when cleanup runs", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		localStorage := tm.createLocalStorage(t, "Recovered")
		tm.app.WriteUnpublishedFile(t, localStorage.Id, "unpublished.dump")
		tm.app.MarkStorageRecovered(t, localStorage.Id)
		tm.app.Clock.Advance(time.Hour)

		tm.app.RunJobPass(t, "storage_file_deletion")

		assert.True(t, tm.app.IsLocalStorageFilePresent(t, "unpublished.dump"))
	})

	t.Run("refuses every write while the recovered instance is paused", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithRecoveryPause())
		localStorage := tm.createLocalStorage(t, "Ordinary")

		err := tm.app.WriteStorageFile(t.Context(),
			storages.StoredFileReference{StorageID: localStorage.Id, FileName: "new.dump"},
			strings.NewReader("new backup"))

		requireProblemCode(t, err, apierror.CodeRecoveryPaused)
	})
}

func TestConnectionTestsDuringRecoveryPause(t *testing.T) {
	t.Parallel()

	t.Run("refuses saved and unsaved connection tests with recovery_paused", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithRecoveryPause())
		localStorage := tm.createLocalStorage(t, "Recovered")

		savedResponse, err := tm.owner.Client.TestStorageWithResponse(t.Context(), localStorage.Id)
		require.NoError(t, err)
		settingsResponse, err := tm.owner.Client.TestStorageSettingsWithResponse(t.Context(),
			apiclient.StorageSettingsTest{WorkspaceId: tm.workspace.Id, StorageId: nil, Settings: newLocalSettings(t)})
		require.NoError(t, err)

		requireProblem(t, savedResponse.StatusCode(), savedResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryPaused)
		requireProblem(t, settingsResponse.StatusCode(), settingsResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryPaused)
		assertNoRecoveryProbeLeft(t, tm.app)
	})
}

// getRowsWithoutStatus reads the storage row without its validation status and the local provider
// row, so a test can show that validation changed nothing else.
func getRowsWithoutStatus(t *testing.T, app *apptest.App, storageID uuid.UUID) []string {
	t.Helper()

	var rows []string
	app.QuerySQL(t, &rows, `SELECT (row_to_json(s)::jsonb - 'recovery_validation_status')::text
		FROM storages s WHERE id = ?`, storageID)

	var providerRows []string
	app.QuerySQL(t, &providerRows, "SELECT row_to_json(p)::text FROM storages_local p WHERE storage_id = ?", storageID)

	return append(rows, providerRows...)
}

func writeExistingBackupFile(t *testing.T, app *apptest.App) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(app.GetDataFolder(), 0o755))
	path := filepath.Join(app.GetDataFolder(), "existing.dump")
	require.NoError(t, os.WriteFile(path, []byte("existing backup"), 0o600))

	return path
}

func requireProblemCode(t *testing.T, err error, code apierror.Code) {
	t.Helper()

	var problem *apierror.Problem
	require.ErrorAs(t, err, &problem)
	assert.Equal(t, code, problem.Code)
}

func assertNoRecoveryProbeLeft(t *testing.T, app *apptest.App) {
	t.Helper()

	probes, err := filepath.Glob(filepath.Join(app.GetDataFolder(), ".databasus-recovery-probe-*"))
	require.NoError(t, err)
	assert.Empty(t, probes)
}
