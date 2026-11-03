package backups_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

var (
	errDownloadNotServed    = errors.New("the download is not served yet")
	errProgressNotRefreshed = errors.New("no progress refresh recorded the moved duration")
)

// backupSourceDatabase is a database of the team's workspace on a restoreServer whose config backs it
// up into a local storage, with no backup yet.
type backupSourceDatabase struct {
	databaseID uuid.UUID
	sourceName string
	storageID  uuid.UUID
}

func newBackupSourceDatabase(t *testing.T, tm team, server restoreServer) backupSourceDatabase {
	t.Helper()

	sourceName := server.createSourceDatabase(t)
	createdDatabase := apptest.CreateDatabase(t, tm.owner, tm.workspace.Id, server.newDatabaseSettings(t, sourceName))
	storageID := tm.createLocalStorage(t)
	saveLogicalBackupConfig(t, tm.owner, createdDatabase.Id, storageID)

	return backupSourceDatabase{databaseID: createdDatabase.Id, sourceName: sourceName, storageID: storageID}
}

// startBlockedSourceBackup starts a backup while the source's table items is locked, so the backup
// stays in progress until the returned release or the end of the test.
func startBlockedSourceBackup(
	t *testing.T,
	tm team,
	server restoreServer,
	source backupSourceDatabase,
) (apiclient.LogicalBackup, func()) {
	t.Helper()

	releaseLock := server.lockItems(t, source.sourceName)
	startedBackup := startRestorableBackup(t, tm.owner, source.databaseID)
	server.waitUntilSessionWaitsOnLock(t, source.sourceName)

	return startedBackup, releaseLock
}

// startQuietDumpBackup starts a backup whose dump tool waits on a locked view that references no
// table, because locking a view also locks the tables it references. The size query before the dump
// reads only tables, so unlike startBlockedSourceBackup this holds the backup while its dump streams,
// before the tool has written anything.
func startQuietDumpBackup(
	t *testing.T,
	tm team,
	server restoreServer,
	source backupSourceDatabase,
) apiclient.LogicalBackup {
	t.Helper()

	server.exec(t, source.sourceName, "CREATE VIEW quiet_view AS SELECT 1 AS id")
	server.lockRelation(t, source.sourceName, "quiet_view")
	startedBackup := startRestorableBackup(t, tm.owner, source.databaseID)
	server.waitUntilSessionWaitsOnLock(t, source.sourceName)

	return startedBackup
}

// moveClockUntilProgressRefresh moves the clock by a minute and waits for a progress refresh to record
// that duration, which takes up to one real progress interval.
func moveClockUntilProgressRefresh(t *testing.T, tm team, backupID uuid.UUID) apiclient.LogicalBackup {
	t.Helper()

	tm.app.Clock.Advance(time.Minute)

	var refreshedBackup apiclient.LogicalBackup

	testwait.Until(t, "a progress refresh recorded the moved duration", func(context.Context) error {
		refreshedBackup = *requestLogicalBackup(t, tm.owner, backupID).JSON200
		if refreshedBackup.DurationMs < time.Minute.Milliseconds() {
			return fmt.Errorf("%w: %d ms", errProgressNotRefreshed, refreshedBackup.DurationMs)
		}

		return nil
	})

	return refreshedBackup
}

func requestLogicalBackupStart(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
) *apiclient.StartLogicalBackupResponse {
	t.Helper()

	response, err := session.Client.StartLogicalBackupWithResponse(t.Context(),
		apiclient.LogicalBackupStart{DatabaseId: databaseID})
	require.NoError(t, err)

	return response
}

func requestLogicalBackupCancel(
	t *testing.T,
	session apptest.Session,
	backupID uuid.UUID,
) *apiclient.CancelLogicalBackupResponse {
	t.Helper()

	response, err := session.Client.CancelLogicalBackupWithResponse(t.Context(), backupID)
	require.NoError(t, err)

	return response
}

func requestLogicalBackupDeletion(
	t *testing.T,
	session apptest.Session,
	backupID uuid.UUID,
) *apiclient.DeleteLogicalBackupResponse {
	t.Helper()

	response, err := session.Client.DeleteLogicalBackupWithResponse(t.Context(), backupID)
	require.NoError(t, err)

	return response
}

func requestLogicalBackup(
	t *testing.T,
	session apptest.Session,
	backupID uuid.UUID,
) *apiclient.GetLogicalBackupResponse {
	t.Helper()

	response, err := session.Client.GetLogicalBackupWithResponse(t.Context(), backupID)
	require.NoError(t, err)

	return response
}

func newLogicalBackupListParams(databaseID uuid.UUID) apiclient.ListLogicalBackupsParams {
	return apiclient.ListLogicalBackupsParams{
		DatabaseId:    databaseID,
		Status:        nil,
		CreatedBefore: nil,
		Limit:         nil,
		After:         nil,
	}
}

func requestLogicalBackupPage(
	t *testing.T,
	session apptest.Session,
	params apiclient.ListLogicalBackupsParams,
) *apiclient.ListLogicalBackupsResponse {
	t.Helper()

	response, err := session.Client.ListLogicalBackupsWithResponse(t.Context(), &params)
	require.NoError(t, err)

	return response
}

// getLogicalBackupPage fails the test unless the list answers 200.
func getLogicalBackupPage(
	t *testing.T,
	session apptest.Session,
	params apiclient.ListLogicalBackupsParams,
) apiclient.LogicalBackupPage {
	t.Helper()

	response := requestLogicalBackupPage(t, session, params)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func getPageBackupIDs(page apiclient.LogicalBackupPage) []uuid.UUID {
	backupIDs := make([]uuid.UUID, 0, len(page.Items))
	for _, listedBackup := range page.Items {
		backupIDs = append(backupIDs, listedBackup.Id)
	}

	return backupIDs
}

func requestDownloadToken(
	t *testing.T,
	session apptest.Session,
	backupID uuid.UUID,
) *apiclient.CreateDownloadTokenResponse {
	t.Helper()

	response, err := session.Client.CreateDownloadTokenWithResponse(t.Context(),
		apiclient.DownloadTokenCreate{BackupId: backupID})
	require.NoError(t, err)

	return response
}

// issueDownloadToken fails the test unless the token is issued.
func issueDownloadToken(t *testing.T, session apptest.Session, backupID uuid.UUID) apiclient.DownloadToken {
	t.Helper()

	response := requestDownloadToken(t, session, backupID)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

// requestBackupDownload downloads without a session, as a browser link does, and reads the whole file.
func requestBackupDownload(
	t *testing.T,
	tm team,
	backupID uuid.UUID,
	token string,
) *apiclient.DownloadLogicalBackupResponse {
	t.Helper()

	response, err := tm.app.NewClient(t).DownloadLogicalBackupWithResponse(t.Context(), backupID,
		&apiclient.DownloadLogicalBackupParams{Token: token})
	require.NoError(t, err)

	return response
}

// downloadBackupFile fails the test unless the download returns the file.
func downloadBackupFile(t *testing.T, tm team, backupID uuid.UUID, token string) []byte {
	t.Helper()

	response := requestBackupDownload(t, tm, backupID, token)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.Body
}

// waitUntilBackupDownloads retries the download while the user's previous stream is still being
// released, and returns the file once the token is served.
func waitUntilBackupDownloads(t *testing.T, tm team, backupID uuid.UUID, token string) []byte {
	t.Helper()

	var downloadedFile []byte

	testwait.Until(t, "the download with the token is served", func(context.Context) error {
		response := requestBackupDownload(t, tm, backupID, token)
		if response.StatusCode() != http.StatusOK {
			return fmt.Errorf("%w: answered %d: %s", errDownloadNotServed, response.StatusCode(), response.Body)
		}

		downloadedFile = response.Body

		return nil
	})

	return downloadedFile
}

func openBackupDownloadStream(t *testing.T, tm team, backupID uuid.UUID, token string) *http.Response {
	t.Helper()
	response, err := tm.owner.Client.DownloadLogicalBackup(
		t.Context(),
		backupID,
		&apiclient.DownloadLogicalBackupParams{Token: token},
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// saveLogicalBackupSettings saves a config with the storage and the given settings and encryption, and
// fails the test unless the save succeeds.
func saveLogicalBackupSettings(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	logical apiclient.LogicalBackupSettings,
	encryption apiclient.BackupEncryption,
) apiclient.BackupConfig {
	t.Helper()

	input := newLogicalBackupConfigInput(t, &storageID, logical)
	input.Encryption = encryption

	response := updateBackupConfig(t, session, databaseID, input)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

// insertLogicalBackupRecord writes a finished backup's row made at createdAt directly, because the API
// makes every backup at the current time and a failed one only from a failing database.
func insertLogicalBackupRecord(
	t *testing.T,
	tm team,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	status apiclient.LogicalBackupStatus,
	createdAt time.Time,
) uuid.UUID {
	t.Helper()

	backupID := uuid.New()
	tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		backupID, databaseID, storageID, string(status), "backup-"+backupID.String(), createdAt)

	return backupID
}

// getBackupFileName reads the name of the backup's file, which the API never returns.
func getBackupFileName(t *testing.T, tm team, backupID uuid.UUID) string {
	t.Helper()

	var fileNames []string
	tm.app.QuerySQL(t, &fileNames, "SELECT file_name FROM backups_logical WHERE id = ?", backupID)
	require.Len(t, fileNames, 1)

	return fileNames[0]
}

// countRunningLogicalBackups counts the database's backups in progress through the list.
func countRunningLogicalBackups(t *testing.T, session apptest.Session, databaseID uuid.UUID) int {
	t.Helper()

	params := newLogicalBackupListParams(databaseID)
	params.Status = &[]apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusINPROGRESS}

	return len(getLogicalBackupPage(t, session, params).Items)
}
