package backups_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const restoreServerImage = "postgres:17"

var (
	errRestoreNotInStatus      = errors.New("the restore has not reached the status")
	errLogicalBackupNotFinal   = errors.New("the logical backup has not reached the status")
	errNoSessionWaitsOnLock    = errors.New("no session of the database waits on a lock")
	errLogRecordNotWritten     = errors.New("the log record is not written yet")
	errRestoreArchiveNotCopied = errors.New("the restore has not started to copy the archive")
)

// restoreServer is a PostgreSQL server that holds the databases a test backs up and the target
// databases it restores into. Every database on it gets a unique name, because the subtests of a
// test share the server.
type restoreServer struct {
	endpoint containers.Endpoint
}

func startRestoreServer(t *testing.T, image string) restoreServer {
	t.Helper()

	return restoreServer{endpoint: containers.StartPostgres(t, image)}
}

func (s restoreServer) connect(t *testing.T, databaseName string) *pgx.Conn {
	t.Helper()

	return s.connectAs(t, databaseName, containers.PostgresUsername, containers.PostgresPassword)
}

func (s restoreServer) connectAs(t *testing.T, databaseName, username, password string) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), fmt.Sprintf(
		"host=%s port=%d user='%s' password='%s' dbname=%s sslmode=disable",
		s.endpoint.Host, s.endpoint.Port, username, password, databaseName,
	))
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(t.Context())) })

	return conn
}

func (s restoreServer) exec(t *testing.T, databaseName, statement string) {
	t.Helper()

	_, err := s.connect(t, databaseName).Exec(t.Context(), statement)
	require.NoError(t, err)
}

// createDatabase creates an empty database with a unique name and returns the name.
func (s restoreServer) createDatabase(t *testing.T, prefix string) string {
	t.Helper()

	databaseName := prefix + "_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	s.exec(t, containers.PostgresDatabase, "CREATE DATABASE "+databaseName)

	return databaseName
}

// createSourceDatabase creates a database with the table items of three rows, which a backup dumps.
func (s restoreServer) createSourceDatabase(t *testing.T) string {
	t.Helper()

	databaseName := s.createDatabase(t, "source")
	s.exec(t, databaseName, "CREATE TABLE items (id INT PRIMARY KEY, name TEXT NOT NULL)")
	s.exec(t, databaseName, "INSERT INTO items VALUES (1, 'first'), (2, 'second'), (3, 'third')")

	return databaseName
}

func (s restoreServer) countItems(t *testing.T, databaseName string) int {
	t.Helper()

	var count int
	require.NoError(t, s.connect(t, databaseName).QueryRow(t.Context(), "SELECT COUNT(*) FROM items").Scan(&count))

	return count
}

// lockItems holds the table items of the database in ACCESS EXCLUSIVE mode until the returned release
// or the end of the test, so a dump or a restore that touches the table waits for it.
func (s restoreServer) lockItems(t *testing.T, databaseName string) (release func()) {
	t.Helper()

	return s.lockRelation(t, databaseName, "items")
}

func (s restoreServer) lockRelation(t *testing.T, databaseName, relationName string) (release func()) {
	t.Helper()

	transaction, err := s.connect(t, databaseName).Begin(t.Context())
	require.NoError(t, err)

	_, err = transaction.Exec(t.Context(), "LOCK TABLE "+relationName+" IN ACCESS EXCLUSIVE MODE")
	require.NoError(t, err)

	release = func() { _ = transaction.Rollback(context.WithoutCancel(t.Context())) }
	t.Cleanup(release)

	return release
}

func (s restoreServer) waitUntilSessionWaitsOnLock(t *testing.T, databaseName string) {
	t.Helper()

	adminConn := s.connect(t, containers.PostgresDatabase)

	testwait.Until(t, "a session of "+databaseName+" waits on a lock", func(ctx context.Context) error {
		var waitingCount int

		err := adminConn.QueryRow(ctx,
			"SELECT COUNT(*) FROM pg_stat_activity WHERE datname = $1 AND wait_event_type = 'Lock'",
			databaseName).Scan(&waitingCount)
		if err != nil {
			return err
		}

		if waitingCount == 0 {
			return errNoSessionWaitsOnLock
		}

		return nil
	})
}

func (s restoreServer) hasItemsTable(t *testing.T, databaseName string) bool {
	t.Helper()

	var isPresent bool
	require.NoError(t, s.connect(t, databaseName).
		QueryRow(t.Context(), "SELECT to_regclass('public.items') IS NOT NULL").Scan(&isPresent))

	return isPresent
}

func (s restoreServer) newDatabaseSettings(t *testing.T, databaseName string) apiclient.DatabaseSettings {
	t.Helper()

	var settings apiclient.DatabaseSettings
	require.NoError(t, settings.FromPostgresqlLogicalSettings(apiclient.PostgresqlLogicalSettings{
		Type:               apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
		Host:               s.endpoint.Host,
		Port:               s.endpoint.Port,
		Username:           containers.PostgresUsername,
		Password:           new(containers.PostgresPassword),
		Database:           new(databaseName),
		SslMode:            apiclient.PostgresqlSslModeDisable,
		SslClientCert:      nil,
		SslClientKey:       nil,
		SslRootCert:        nil,
		SshTunnel:          newDisabledRestoreSSHTunnel(),
		IncludeSchemas:     []string{},
		ExcludeTables:      []string{},
		ExcludeTableData:   []string{},
		CpuCount:           1,
		IsSkipUserMappings: false,
		Version:            nil,
	}))

	return settings
}

// restoreTargetSpec is a target database of a restore on a restoreServer.
type restoreTargetSpec struct {
	databaseName string
	username     string
	password     string
	// cpuCount above 1 makes the restore download the archive into a file first.
	cpuCount int
}

// newTargetSpec creates an empty target database reached as the server's superuser.
func (s restoreServer) newTargetSpec(t *testing.T) restoreTargetSpec {
	t.Helper()

	return restoreTargetSpec{
		databaseName: s.createDatabase(t, "target"),
		username:     containers.PostgresUsername,
		password:     containers.PostgresPassword,
		cpuCount:     1,
	}
}

// newBlockedTargetSpec creates a target database whose table items is locked, so a restore into it
// runs until the lock is released.
func (s restoreServer) newBlockedTargetSpec(t *testing.T) (restoreTargetSpec, func()) {
	t.Helper()

	targetSpec := s.newTargetSpec(t)
	s.exec(t, targetSpec.databaseName, "CREATE TABLE items (id INT PRIMARY KEY, name TEXT NOT NULL)")

	return targetSpec, s.lockItems(t, targetSpec.databaseName)
}

func (s restoreServer) newRestoreTarget(t *testing.T, targetSpec restoreTargetSpec) apiclient.RestoreTarget {
	t.Helper()

	return newPostgresRestoreTarget(t, s.endpoint, targetSpec)
}

func newPostgresRestoreTarget(
	t *testing.T,
	endpoint containers.Endpoint,
	targetSpec restoreTargetSpec,
) apiclient.RestoreTarget {
	t.Helper()

	var target apiclient.RestoreTarget
	require.NoError(t, target.FromPostgresqlRestoreTarget(apiclient.PostgresqlRestoreTarget{
		Type:                apiclient.PostgresqlRestoreTargetTypePOSTGRESLOGICAL,
		Host:                endpoint.Host,
		Port:                endpoint.Port,
		Username:            targetSpec.username,
		Password:            new(targetSpec.password),
		Database:            targetSpec.databaseName,
		SslMode:             apiclient.PostgresqlSslModeDisable,
		SslClientCert:       nil,
		SslClientKey:        nil,
		SslRootCert:         nil,
		SshTunnel:           newDisabledRestoreSSHTunnel(),
		CpuCount:            targetSpec.cpuCount,
		IsExcludeExtensions: false,
		IsRestoreOwnership:  false,
		IsRestorePrivileges: false,
	}))

	return target
}

func newDisabledRestoreSSHTunnel() apiclient.SshTunnelSettings {
	return apiclient.SshTunnelSettings{
		IsEnabled:            false,
		AuthType:             nil,
		Host:                 nil,
		Port:                 nil,
		Username:             nil,
		Password:             nil,
		PrivateKey:           nil,
		PrivateKeyPassphrase: nil,
	}
}

// backedUpDatabase is a database of the team's workspace on a restoreServer with a completed backup.
type backedUpDatabase struct {
	databaseID   uuid.UUID
	sourceName   string
	storageID    uuid.UUID
	backup       apiclient.LogicalBackup
	backupConfig apiclient.BackupConfig
}

// newBackedUpDatabase creates the database through the API, saves its config with a local storage and
// waits for its first backup to complete.
func newBackedUpDatabase(t *testing.T, tm team, server restoreServer) backedUpDatabase {
	t.Helper()

	sourceName := server.createSourceDatabase(t)
	createdDatabase := apptest.CreateDatabase(t, tm.owner, tm.workspace.Id, server.newDatabaseSettings(t, sourceName))
	storageID := tm.createLocalStorage(t)
	backupConfig := saveLogicalBackupConfig(t, tm.owner, createdDatabase.Id, storageID)

	return backedUpDatabase{
		databaseID:   createdDatabase.Id,
		sourceName:   sourceName,
		storageID:    storageID,
		backup:       completeRestorableBackup(t, tm.owner, createdDatabase.Id),
		backupConfig: backupConfig,
	}
}

// completeRestorableBackup starts a backup of the database and waits until it completes.
func completeRestorableBackup(t *testing.T, session apptest.Session, databaseID uuid.UUID) apiclient.LogicalBackup {
	t.Helper()

	startedBackup := startRestorableBackup(t, session, databaseID)

	return waitForRestorableBackupStatus(t, session, startedBackup.Id, apiclient.LogicalBackupStatusCOMPLETED)
}

func startRestorableBackup(t *testing.T, session apptest.Session, databaseID uuid.UUID) apiclient.LogicalBackup {
	t.Helper()

	response, err := session.Client.StartLogicalBackupWithResponse(t.Context(),
		apiclient.LogicalBackupStart{DatabaseId: databaseID})
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))

	return *response.JSON202
}

// startBlockedBackup starts a backup of the database while its table items is locked, so the backup
// stays in progress until the test ends.
func startBlockedBackup(
	t *testing.T,
	tm team,
	server restoreServer,
	database backedUpDatabase,
) apiclient.LogicalBackup {
	t.Helper()

	server.lockItems(t, database.sourceName)
	startedBackup := startRestorableBackup(t, tm.owner, database.databaseID)
	server.waitUntilSessionWaitsOnLock(t, database.sourceName)

	return startedBackup
}

func waitForRestorableBackupStatus(
	t *testing.T,
	session apptest.Session,
	backupID uuid.UUID,
	status apiclient.LogicalBackupStatus,
) apiclient.LogicalBackup {
	t.Helper()

	var readBackup apiclient.LogicalBackup

	testwait.Until(t, "the logical backup becomes "+string(status), func(ctx context.Context) error {
		response, err := session.Client.GetLogicalBackupWithResponse(ctx, backupID)
		if err != nil {
			return err
		}

		if response.JSON200 == nil {
			return fmt.Errorf("%w: answered %d: %s", errLogicalBackupNotFinal, response.StatusCode(), response.Body)
		}

		readBackup = *response.JSON200
		if readBackup.Status != status {
			if readBackup.Status != apiclient.LogicalBackupStatusINPROGRESS {
				require.Equal(
					t,
					status,
					readBackup.Status,
					"the logical backup reached a terminal status: %v",
					readBackup.FailMessage,
				)
			}
			return fmt.Errorf("%w: %s, fail message %v", errLogicalBackupNotFinal, readBackup.Status,
				readBackup.FailMessage)
		}

		return nil
	})

	return readBackup
}

func getLogicalBackupStatus(t *testing.T, session apptest.Session, backupID uuid.UUID) apiclient.LogicalBackupStatus {
	t.Helper()

	response, err := session.Client.GetLogicalBackupWithResponse(t.Context(), backupID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.JSON200.Status
}

func listLogicalBackupIDs(t *testing.T, session apptest.Session, databaseID uuid.UUID) []uuid.UUID {
	t.Helper()

	response, err := session.Client.ListLogicalBackupsWithResponse(t.Context(), &apiclient.ListLogicalBackupsParams{
		DatabaseId:    databaseID,
		Status:        nil,
		CreatedBefore: nil,
		Limit:         nil,
		After:         nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	backupIDs := make([]uuid.UUID, 0, len(response.JSON200.Items))
	for _, listedBackup := range response.JSON200.Items {
		backupIDs = append(backupIDs, listedBackup.Id)
	}

	return backupIDs
}

func startRestore(
	t *testing.T,
	session apptest.Session,
	backupID uuid.UUID,
	target apiclient.RestoreTarget,
) *apiclient.StartRestoreResponse {
	t.Helper()

	response, err := session.Client.StartRestoreWithResponse(t.Context(),
		apiclient.RestoreStart{BackupId: backupID, Target: target})
	require.NoError(t, err)

	return response
}

func requireStartedRestore(
	t *testing.T,
	session apptest.Session,
	backupID uuid.UUID,
	target apiclient.RestoreTarget,
) apiclient.Restore {
	t.Helper()

	response := startRestore(t, session, backupID, target)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
	require.Equal(t, apiclient.RestoreStatusINPROGRESS, response.JSON202.Status)

	return *response.JSON202
}

// startBlockedRestore starts a restore of the backup into a target whose table items is locked, and
// waits until the restore tool waits for the lock, so the restore runs until the test ends or cancels it.
func startBlockedRestore(
	t *testing.T,
	session apptest.Session,
	server restoreServer,
	backupID uuid.UUID,
) (apiclient.Restore, restoreTargetSpec) {
	t.Helper()

	targetSpec, _ := server.newBlockedTargetSpec(t)
	startedRestore := requireStartedRestore(t, session, backupID, server.newRestoreTarget(t, targetSpec))
	server.waitUntilSessionWaitsOnLock(t, targetSpec.databaseName)

	return startedRestore, targetSpec
}

// runningWork is a database of the team with a completed backup, a backup that runs and a restore of
// the completed backup that runs, both held by locks until the test ends.
type runningWork struct {
	database       backedUpDatabase
	runningBackup  apiclient.LogicalBackup
	runningRestore apiclient.Restore
}

func startRunningWork(t *testing.T, tm team, server restoreServer) runningWork {
	t.Helper()

	database := newBackedUpDatabase(t, tm, server)
	runningRestore, _ := startBlockedRestore(t, tm.owner, server, database.backup.Id)

	return runningWork{
		database:       database,
		runningBackup:  startBlockedBackup(t, tm, server, database),
		runningRestore: runningRestore,
	}
}

func waitForRestoreStatus(
	t *testing.T,
	session apptest.Session,
	restoreID uuid.UUID,
	status apiclient.RestoreStatus,
) apiclient.Restore {
	t.Helper()

	var readRestore apiclient.Restore

	testwait.Until(t, "the restore becomes "+string(status), func(ctx context.Context) error {
		response, err := session.Client.GetRestoreWithResponse(ctx, restoreID)
		if err != nil {
			return err
		}

		if response.JSON200 == nil {
			return fmt.Errorf("%w: answered %d: %s", errRestoreNotInStatus, response.StatusCode(), response.Body)
		}

		readRestore = *response.JSON200
		if readRestore.Status != status {
			return fmt.Errorf("%w: %s, fail message %v", errRestoreNotInStatus, readRestore.Status,
				readRestore.FailMessage)
		}

		return nil
	})

	return readRestore
}

func getRestore(t *testing.T, session apptest.Session, restoreID uuid.UUID) apiclient.Restore {
	t.Helper()

	response, err := session.Client.GetRestoreWithResponse(t.Context(), restoreID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func cancelRestore(t *testing.T, session apptest.Session, restoreID uuid.UUID) *apiclient.CancelRestoreResponse {
	t.Helper()

	response, err := session.Client.CancelRestoreWithResponse(t.Context(), restoreID)
	require.NoError(t, err)

	return response
}

func listRestores(t *testing.T, session apptest.Session, backupID uuid.UUID) []apiclient.Restore {
	t.Helper()

	response, err := session.Client.ListRestoresWithResponse(t.Context(),
		&apiclient.ListRestoresParams{BackupId: backupID, Limit: nil, After: nil})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.JSON200.Items
}

func countRestoresOfBackup(t *testing.T, tm team, backupID uuid.UUID) int {
	t.Helper()

	return countRows(t, tm, "SELECT COUNT(*) FROM backups_restores WHERE backup_id = ?", backupID)
}

// getBackupFilePath is where the local storage keeps the file of the backup. The API never returns
// the file name.
func getBackupFilePath(t *testing.T, tm team, backupID uuid.UUID) string {
	t.Helper()

	var fileNames []string
	tm.app.QuerySQL(t, &fileNames, "SELECT file_name FROM backups_logical WHERE id = ?", backupID)
	require.Len(t, fileNames, 1)

	return tm.app.GetLocalStorageFilePath(fileNames[0])
}

// listRestoreToolProcesses lists the running pg_restore processes that restore into the database. The
// server's own backends for the database are not among them, because their first argument is not
// pg_restore.
func listRestoreToolProcesses(t *testing.T, databaseName string) []string {
	t.Helper()

	processDirs, err := filepath.Glob("/proc/[0-9]*")
	require.NoError(t, err)

	var restoreToolPIDs []string

	for _, processDir := range processDirs {
		commandLine, err := os.ReadFile(filepath.Join(processDir, "cmdline"))
		if err != nil {
			continue
		}

		args := strings.Split(strings.TrimRight(string(commandLine), "\x00"), "\x00")
		if filepath.Base(args[0]) == "pg_restore" && slices.Contains(args, databaseName) {
			restoreToolPIDs = append(restoreToolPIDs, filepath.Base(processDir))
		}
	}

	return restoreToolPIDs
}

// listRestoreArchiveDirs lists the folders in which restores download their archives.
func listRestoreArchiveDirs(t *testing.T, tm team) []string {
	t.Helper()

	archiveDirs, err := filepath.Glob(filepath.Join(tm.app.GetTempFolder(), "restore_*"))
	require.NoError(t, err)

	return archiveDirs
}

func readApplicationLogs(t *testing.T, tm team) string {
	t.Helper()

	logPaths, err := filepath.Glob(filepath.Join(filepath.Dir(tm.app.GetLocalStorageFilePath("")), "databasus*.log"))
	require.NoError(t, err)

	var logs strings.Builder

	for _, logPath := range logPaths {
		logContent, err := os.ReadFile(logPath)
		require.NoError(t, err)

		logs.Write(logContent)
	}

	return logs.String()
}

type restoreLogRecord struct {
	message   string
	restoreID uuid.UUID
}

// The file sink writes from a queue, so a record reaches the file a moment after the call that logs it.
func waitForRestoreLogRecord(t *testing.T, tm team, record restoreLogRecord) {
	t.Helper()

	testwait.Until(t, "the log record "+record.message, func(context.Context) error {
		for line := range strings.SplitSeq(readApplicationLogs(t, tm), "\n") {
			if strings.Contains(line, record.message) && strings.Contains(line, record.restoreID.String()) {
				return nil
			}
		}

		return errLogRecordNotWritten
	})
}
