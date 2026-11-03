package backups_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one source server.
func TestStartLogicalBackup(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns 202 with a listed backup in progress that later completes with its size", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		source := newBackupSourceDatabase(t, tm, server)

		response := requestLogicalBackupStart(t, member, source.databaseID)

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		startedBackup := *response.JSON202
		assert.Equal(t, apiclient.LogicalBackupStatusINPROGRESS, startedBackup.Status)
		assert.Equal(t, source.databaseID, startedBackup.DatabaseId)
		assert.Equal(t, source.storageID, startedBackup.StorageId)
		assert.Contains(t, listLogicalBackupIDs(t, member, source.databaseID), startedBackup.Id)

		completedBackup := waitForRestorableBackupStatus(t, member, startedBackup.Id,
			apiclient.LogicalBackupStatusCOMPLETED)
		assert.Positive(t, completedBackup.SizeMb)
		assert.Nil(t, completedBackup.FailMessage)
		assert.Equal(t, apiclient.BackupEncryptionNONE, completedBackup.Encryption)

		issuedToken := issueDownloadToken(t, tm.owner, startedBackup.Id)
		downloadedFile := downloadBackupFile(t, tm, startedBackup.Id, issuedToken.Token)
		assert.InDelta(t, float64(len(downloadedFile))/(1024*1024), completedBackup.SizeMb, 1e-9)
	})

	t.Run("advances the duration and keeps the size while the dump produces no output", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		runningBackup := startQuietDumpBackup(t, tm, server, source)

		backupBeforeRefresh := requestLogicalBackup(t, tm.owner, runningBackup.Id).JSON200
		assert.Equal(t, apiclient.LogicalBackupStatusINPROGRESS, backupBeforeRefresh.Status)
		assert.Zero(t, backupBeforeRefresh.SizeMb)
		assert.Zero(t, backupBeforeRefresh.DurationMs)

		refreshedBackup := moveClockUntilProgressRefresh(t, tm, runningBackup.Id)

		assert.Equal(t, apiclient.LogicalBackupStatusINPROGRESS, refreshedBackup.Status)
		assert.Zero(t, refreshedBackup.SizeMb)
	})

	t.Run("returns 409 already_running to every concurrent start but one", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		server.lockItems(t, source.sourceName)

		const startCount = 4

		startStatuses := make([]int, startCount)
		startBodies := make([][]byte, startCount)
		startSignal := make(chan struct{})

		var startsDone sync.WaitGroup

		for startIndex := range startCount {
			startsDone.Go(func() {
				<-startSignal

				response, err := tm.owner.Client.StartLogicalBackupWithResponse(t.Context(),
					apiclient.LogicalBackupStart{DatabaseId: source.databaseID})
				if err != nil {
					t.Errorf("start a backup: %v", err)

					return
				}

				startStatuses[startIndex] = response.StatusCode()
				startBodies[startIndex] = response.Body
			})
		}

		close(startSignal)
		startsDone.Wait()

		acceptedCount := 0

		for startIndex, startStatus := range startStatuses {
			if startStatus == http.StatusAccepted {
				acceptedCount++

				continue
			}

			requireProblem(t, startStatus, startBodies[startIndex], http.StatusConflict,
				apiclient.ProblemCodeAlreadyRunning)
		}

		assert.Equal(t, 1, acceptedCount)
		assert.Equal(t, 1, countRunningLogicalBackups(t, tm.owner, source.databaseID))
		assert.Len(t, listLogicalBackupIDs(t, tm.owner, source.databaseID), 1)
	})

	t.Run("returns 409 already_running when a backup of the database is running", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		runningBackup, _ := startBlockedSourceBackup(t, tm, server, source)

		response := requestLogicalBackupStart(t, tm.owner, source.databaseID)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeAlreadyRunning)
		assert.Equal(t, 1, countRunningLogicalBackups(t, tm.owner, source.databaseID))
		assert.Equal(t, []uuid.UUID{runningBackup.Id}, listLogicalBackupIDs(t, tm.owner, source.databaseID))
	})

	t.Run("returns 400 invalid_request when the config names no storage and lists no backup", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		input := newLogicalBackupConfigInput(t, nil, newLogicalBackupSettings())
		input.IsBackupsEnabled = false
		saveResponse := updateBackupConfig(t, tm.owner, databaseID, input)
		require.Equal(t, http.StatusOK, saveResponse.StatusCode(), string(saveResponse.Body))

		response := requestLogicalBackupStart(t, tm.owner, databaseID)

		requireInvalidField(t, response.StatusCode(), response.Body, "databaseId")
		assert.Empty(t, listLogicalBackupIDs(t, tm.owner, databaseID))
	})

	t.Run("returns 404 not_found for a database that does not exist", func(t *testing.T) {
		tm := newTeam(t)

		response := requestLogicalBackupStart(t, tm.owner, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 401 unauthenticated without a session and starts nothing", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)

		response, err := tm.app.NewClient(t).StartLogicalBackupWithResponse(t.Context(),
			apiclient.LogicalBackupStart{DatabaseId: source.databaseID})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.Empty(t, listLogicalBackupIDs(t, tm.owner, source.databaseID))
	})
}

//nolint:tparallel // The subtests share one physical PostgreSQL server.
func TestStartLogicalBackupOfPhysicalDatabase(t *testing.T) {
	t.Parallel()

	physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")

	t.Run("returns 400 invalid_request for a physical database and starts nothing", func(t *testing.T) {
		tm := newTeam(t)
		physicalDatabase := tm.createPhysicalDatabase(t, physicalServer, apiclient.PhysicalBackupTypeFULL)
		savePhysicalBackupConfigWithStorage(t, tm, physicalDatabase.Id, tm.createLocalStorage(t))

		response := requestLogicalBackupStart(t, tm.owner, physicalDatabase.Id)

		requireInvalidField(t, response.StatusCode(), response.Body, "databaseId")
		assert.Zero(t, countRows(t, tm, "SELECT COUNT(*) FROM backups_logical WHERE database_id = ?",
			physicalDatabase.Id))
	})
}
