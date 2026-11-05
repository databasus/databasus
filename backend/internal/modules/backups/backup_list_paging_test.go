package backups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestListLogicalBackups(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("lists only failed backups newest first with the status filter FAILED", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		statuses := []apiclient.LogicalBackupStatus{
			apiclient.LogicalBackupStatusFAILED,
			apiclient.LogicalBackupStatusCOMPLETED,
			apiclient.LogicalBackupStatusFAILED,
			apiclient.LogicalBackupStatusCANCELED,
			apiclient.LogicalBackupStatusFAILED,
		}
		backupIDs := insertBackupsMinutesApart(t, tm, databaseID, storageID, statuses)

		params := newLogicalBackupListParams(databaseID)
		params.Status = &[]apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusFAILED}
		page := getLogicalBackupPage(t, tm.owner, params)

		assert.Equal(t, []uuid.UUID{backupIDs[0], backupIDs[2], backupIDs[4]}, getPageBackupIDs(page))
		assert.Nil(t, page.NextCursor)
	})

	t.Run("shows the status, size, duration, encryption and time of a completed backup", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		page := getLogicalBackupPage(t, tm.owner, newLogicalBackupListParams(database.databaseID))

		require.Len(t, page.Items, 1)
		listedBackup := page.Items[0]
		assert.Equal(t, database.backup.Id, listedBackup.Id)
		assert.Equal(t, apiclient.LogicalBackupStatusCOMPLETED, listedBackup.Status)
		assert.Nil(t, listedBackup.FailMessage)
		assert.Positive(t, listedBackup.SizeMb)
		assert.Equal(t, database.backup.DurationMs, listedBackup.DurationMs)
		assert.Equal(t, apiclient.BackupEncryptionNONE, listedBackup.Encryption)
		assert.WithinDuration(t, database.backup.CreatedAt, listedBackup.CreatedAt, time.Millisecond)
	})

	t.Run("returns 404 not_found for a database that does not exist", func(t *testing.T) {
		tm := newTeam(t)

		response := requestLogicalBackupPage(t, tm.owner, newLogicalBackupListParams(uuid.New()))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

//nolint:tparallel // The subtests share one source server.
func TestListLogicalBackupsWithCursor(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns the next backups right after the last one of the first page", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		backupIDs := insertCompletedBackupsMinutesApart(t, tm, databaseID, tm.createLocalStorage(t), 25)

		params := newLogicalBackupListParams(databaseID)
		params.Limit = new(20)
		firstPage := getLogicalBackupPage(t, tm.owner, params)

		require.NotNil(t, firstPage.NextCursor)
		assert.Equal(t, backupIDs[:20], getPageBackupIDs(firstPage))

		params.After = firstPage.NextCursor
		secondPage := getLogicalBackupPage(t, tm.owner, params)

		assert.Equal(t, backupIDs[20:], getPageBackupIDs(secondPage))
		assert.Nil(t, secondPage.NextCursor)
	})

	t.Run("neither repeats nor skips a backup when a backup starts between the requests", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		backupIDs := insertCompletedBackupsMinutesApart(t, tm, source.databaseID, source.storageID, 4)

		params := newLogicalBackupListParams(source.databaseID)
		params.Limit = new(2)
		firstPage := getLogicalBackupPage(t, tm.owner, params)
		startedBackup := startRestorableBackup(t, tm.owner, source.databaseID)

		params.After = firstPage.NextCursor
		secondPage := getLogicalBackupPage(t, tm.owner, params)

		assert.Equal(t, backupIDs[:2], getPageBackupIDs(firstPage))
		assert.Equal(t, backupIDs[2:], getPageBackupIDs(secondPage))
		assert.NotContains(t, getPageBackupIDs(secondPage), startedBackup.Id)
		waitForRestorableBackupStatus(t, tm.owner, startedBackup.Id, apiclient.LogicalBackupStatusCOMPLETED)
	})

	t.Run("applies the status filter to every page", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		statuses := []apiclient.LogicalBackupStatus{
			apiclient.LogicalBackupStatusFAILED,
			apiclient.LogicalBackupStatusCOMPLETED,
			apiclient.LogicalBackupStatusFAILED,
			apiclient.LogicalBackupStatusCOMPLETED,
			apiclient.LogicalBackupStatusFAILED,
			apiclient.LogicalBackupStatusCANCELED,
			apiclient.LogicalBackupStatusFAILED,
		}
		backupIDs := insertBackupsMinutesApart(t, tm, databaseID, tm.createLocalStorage(t), statuses)

		params := newLogicalBackupListParams(databaseID)
		params.Status = &[]apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusFAILED}
		params.Limit = new(2)
		firstPage := getLogicalBackupPage(t, tm.owner, params)

		require.NotNil(t, firstPage.NextCursor)
		assert.Equal(t, []uuid.UUID{backupIDs[0], backupIDs[2]}, getPageBackupIDs(firstPage))

		params.After = firstPage.NextCursor
		secondPage := getLogicalBackupPage(t, tm.owner, params)

		assert.Equal(t, []uuid.UUID{backupIDs[4], backupIDs[6]}, getPageBackupIDs(secondPage))
	})

	t.Run("applies the time filter to every page", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		backupIDs := insertCompletedBackupsMinutesApart(t, tm, databaseID, tm.createLocalStorage(t), 7)
		thirdBackup := getLogicalBackupPage(t, tm.owner, newLogicalBackupListParams(databaseID)).Items[2]

		params := newLogicalBackupListParams(databaseID)
		params.CreatedBefore = &thirdBackup.CreatedAt
		params.Limit = new(2)
		firstPage := getLogicalBackupPage(t, tm.owner, params)

		require.NotNil(t, firstPage.NextCursor)
		assert.Equal(t, backupIDs[3:5], getPageBackupIDs(firstPage))

		params.After = firstPage.NextCursor
		secondPage := getLogicalBackupPage(t, tm.owner, params)

		assert.Equal(t, backupIDs[5:7], getPageBackupIDs(secondPage))
		assert.Nil(t, secondPage.NextCursor)
	})

	t.Run("carries no cursor on the page that holds the oldest backup", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		backupIDs := insertCompletedBackupsMinutesApart(t, tm, databaseID, tm.createLocalStorage(t), 3)

		params := newLogicalBackupListParams(databaseID)
		params.Limit = new(3)
		page := getLogicalBackupPage(t, tm.owner, params)

		assert.Equal(t, backupIDs, getPageBackupIDs(page))
		assert.Nil(t, page.NextCursor)
	})

	t.Run("returns 400 invalid_request with no rows for a cursor the server did not issue", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		insertCompletedBackupsMinutesApart(t, tm, databaseID, tm.createLocalStorage(t), 3)

		params := newLogicalBackupListParams(databaseID)
		params.After = new("not-a-cursor")
		response := requestLogicalBackupPage(t, tm.owner, params)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.Nil(t, response.JSON200)
		assert.NotContains(t, string(response.Body), "items")
	})
}

// insertBackupsMinutesApart writes one finished backup per status, the first one the newest, a minute
// apart and older than the clock, and returns their IDs newest first.
func insertBackupsMinutesApart(
	t *testing.T,
	tm team,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	statuses []apiclient.LogicalBackupStatus,
) []uuid.UUID {
	t.Helper()

	newestTime := tm.app.Clock.Now().Add(-time.Minute)
	backupIDs := make([]uuid.UUID, 0, len(statuses))

	for statusIndex, status := range statuses {
		createdAt := newestTime.Add(-time.Duration(statusIndex) * time.Minute)
		backupIDs = append(backupIDs, insertLogicalBackupRecord(t, tm, databaseID, storageID, status, createdAt))
	}

	return backupIDs
}

func insertCompletedBackupsMinutesApart(
	t *testing.T,
	tm team,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	count int,
) []uuid.UUID {
	t.Helper()

	statuses := make([]apiclient.LogicalBackupStatus, count)
	for statusIndex := range statuses {
		statuses[statusIndex] = apiclient.LogicalBackupStatusCOMPLETED
	}

	return insertBackupsMinutesApart(t, tm, databaseID, storageID, statuses)
}
