package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/process"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

// The markers the physical restore set recovers to the point in time: every phase before the
// target, and not the row written after it.
const (
	markerBeforeFull   = "before-full"
	markerAfterFull    = "after-full"
	markerBeforeTarget = "before-target"
	markerAfterTarget  = "after-target"
	walActivityBytes   = 32 * 1024 * 1024
)

func getMarkersAtTarget() []string {
	return []string{markerBeforeFull, markerAfterFull, markerBeforeTarget}
}

// physicalChain is the FULL, the INCREMENTAL and the streamed WAL the source installation catalogs
// before its Databasus backup, with the point in time a restore targets.
type physicalChain struct {
	databaseID uuid.UUID
	targetTime time.Time
}

func buildPhysicalChain(
	t *testing.T,
	cycle *recoveryCycle,
	owner accountSession,
	workspaceID, storageID uuid.UUID,
) physicalChain {
	t.Helper()

	connection := cycle.openSourceDatabase(t, sourceDatabaseName)
	_, err := connection.Exec(t.Context(), "CREATE TABLE "+physicalMarkerTable+" (phase TEXT NOT NULL)")
	require.NoError(t, err)
	insertMarker(t, connection, markerBeforeFull)

	database := createDatabase(t, owner, workspaceID, newPhysicalSourceSettings(t), []uuid.UUID{})
	enablePhysicalBackups(t, owner, database.Id, storageID)
	fullBackup := runPhysicalBackup(t, owner, database.Id, apiclient.PhysicalBackupStartKindFULL)

	insertMarker(t, connection, markerAfterFull)
	generateWalActivity(t, connection)
	_, err = connection.Exec(t.Context(), "CHECKPOINT")
	require.NoError(t, err)
	switchWal(t, connection)
	waitForWalSummary(t, connection, fullBackup.StopLsn)
	incrementalBackup := runPhysicalBackup(t, owner, database.Id, apiclient.PhysicalBackupStartKindINCREMENTAL)
	require.Equal(t, fullBackup.Id, *incrementalBackup.RootFullBackupId)

	insertMarker(t, connection, markerBeforeTarget)
	generateWalActivity(t, connection)
	targetTime := getTimeWithMargin(t, connection)
	insertMarker(t, connection, markerAfterTarget)
	generateWalActivity(t, connection)
	switchWal(t, connection)

	chain := physicalChain{databaseID: database.Id, targetTime: targetTime}
	testwait.Within(backupLimit).Until(t, "the streamed WAL makes the target time restorable",
		func(ctx context.Context) error {
			response, err := requestPointInTimeToken(ctx, owner, chain)
			if err != nil {
				return err
			}

			if response.StatusCode() != http.StatusCreated {
				return fmt.Errorf("the restore token answered %d: %s", response.StatusCode(), response.Body)
			}

			return nil
		})

	return chain
}

func insertMarker(t *testing.T, connection *pgx.Conn, phase string) {
	t.Helper()

	_, err := connection.Exec(t.Context(), "INSERT INTO "+physicalMarkerTable+" (phase) VALUES ($1)", phase)
	require.NoError(t, err)
}

func generateWalActivity(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	_, err := connection.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS recovery_wal_activity (payload TEXT)")
	require.NoError(t, err)

	startLSN := getCurrentLSN(t, connection)
	for getCurrentLSN(t, connection)-startLSN < walActivityBytes {
		_, err = connection.Exec(t.Context(),
			"INSERT INTO recovery_wal_activity SELECT repeat('x', 1024) FROM generate_series(1, 1000)")
		require.NoError(t, err)
	}

	_, err = connection.Exec(t.Context(), "TRUNCATE recovery_wal_activity")
	require.NoError(t, err)
}

func getCurrentLSN(t *testing.T, connection *pgx.Conn) uint64 {
	t.Helper()

	var lsn uint64
	require.NoError(t, connection.QueryRow(t.Context(),
		"SELECT pg_current_wal_lsn() - '0/0'::pg_lsn").Scan(&lsn))

	return lsn
}

// switchWal closes the current WAL segment, so the summarizer and the streamer reach the writes.
func switchWal(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	_, err := connection.Exec(t.Context(), "SELECT pg_switch_wal()")
	require.NoError(t, err)
}

func waitForWalSummary(t *testing.T, connection *pgx.Conn, position string) {
	t.Helper()

	testwait.Until(t, "WAL summaries cover "+position, func(ctx context.Context) error {
		var isSummarized bool

		err := connection.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM pg_available_wal_summaries() "+
				"WHERE start_lsn <= $1::pg_lsn AND end_lsn >= $1::pg_lsn)",
			position).Scan(&isSummarized)
		if err != nil {
			return err
		}

		if !isSummarized {
			return fmt.Errorf("no WAL summary covers %s", position)
		}

		return nil
	})
}

// getTimeWithMargin returns a time of the source server's clock with two seconds of writes-free
// margin on both sides, so the target separates the markers around it.
func getTimeWithMargin(t *testing.T, connection *pgx.Conn) time.Time {
	t.Helper()

	waitForSourceTime(t, connection, getSourceTime(t, connection).Add(2*time.Second))
	targetTime := getSourceTime(t, connection)
	waitForSourceTime(t, connection, targetTime.Add(2*time.Second))

	return targetTime
}

func getSourceTime(t *testing.T, connection *pgx.Conn) time.Time {
	t.Helper()

	var sourceTime time.Time
	require.NoError(t, connection.QueryRow(t.Context(), "SELECT clock_timestamp()").Scan(&sourceTime))

	return sourceTime.UTC()
}

func waitForSourceTime(t *testing.T, connection *pgx.Conn, awaitedTime time.Time) {
	t.Helper()

	testwait.Until(t, "the source clock passes the margin", func(context.Context) error {
		if getSourceTime(t, connection).Before(awaitedTime) {
			return fmt.Errorf("the source clock has not reached %s", awaitedTime)
		}

		return nil
	})
}

func enablePhysicalBackups(t *testing.T, owner accountSession, databaseID, storageID uuid.UUID) {
	t.Helper()

	// The schedules are due once a year, so only the requested backups run during the test.
	cronExpression := "0 0 1 1 *"
	schedule := apiclient.Schedule{Type: apiclient.ScheduleTypeCRON, CronExpression: &cronExpression}

	var settings apiclient.BackupSettings
	require.NoError(t, settings.FromPhysicalBackupSettings(apiclient.PhysicalBackupSettings{
		Type:                      apiclient.PhysicalBackupSettingsTypePHYSICAL,
		FullBackupSchedule:        schedule,
		IncrementalBackupSchedule: &schedule,
		Retention: apiclient.PhysicalRetention{
			Policy:      apiclient.PhysicalRetentionPolicyCHAINS,
			ChainsCount: 50,
		},
		NotificationTriggers: []apiclient.NotificationTrigger{},
		WalLagThresholdBytes: 64 * 1024 * 1024,
	}))

	response, err := owner.Client.UpdateBackupConfigWithResponse(t.Context(), databaseID, apiclient.BackupConfigInput{
		IsBackupsEnabled: true,
		StorageId:        &storageID,
		Encryption:       apiclient.BackupEncryptionNONE,
		Settings:         settings,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

// runPhysicalBackup requests the backup, unless enabling the backups already started one of that
// kind, and waits for it to complete.
func runPhysicalBackup(
	t *testing.T,
	owner accountSession,
	databaseID uuid.UUID,
	kind apiclient.PhysicalBackupStartKind,
) apiclient.PhysicalBackupListRow {
	t.Helper()

	startedAt := time.Now().Add(-time.Minute)
	response, err := owner.Client.StartPhysicalBackupWithResponse(t.Context(),
		apiclient.PhysicalBackupStart{DatabaseId: databaseID, Kind: kind})
	require.NoError(t, err)
	require.Contains(t, []int{http.StatusAccepted, http.StatusConflict}, response.StatusCode(), string(response.Body))

	return waitForPhysicalBackup(t, owner, databaseID, physicalBackupQuery{
		kind: apiclient.PhysicalBackupListRowType(kind), createdAfter: startedAt,
	})
}

type physicalBackupQuery struct {
	kind         apiclient.PhysicalBackupListRowType
	createdAfter time.Time
}

// waitForPhysicalBackup waits for a completed backup of the kind created after the query's time.
func waitForPhysicalBackup(
	t *testing.T,
	owner accountSession,
	databaseID uuid.UUID,
	query physicalBackupQuery,
) apiclient.PhysicalBackupListRow {
	t.Helper()

	var completedBackup apiclient.PhysicalBackupListRow

	testwait.Within(backupLimit).Until(t, "a physical "+string(query.kind)+" backup completes",
		func(ctx context.Context) error {
			response, err := owner.Client.ListPhysicalBackupsWithResponse(ctx, &apiclient.ListPhysicalBackupsParams{
				DatabaseId: databaseID,
				Type:       &[]apiclient.PhysicalBackupListRowType{query.kind},
				Limit:      new(100),
			})
			if err != nil {
				return err
			}

			if response.StatusCode() != http.StatusOK {
				return fmt.Errorf("list the physical backups answered %d", response.StatusCode())
			}

			for _, backup := range response.JSON200.Items {
				if backup.CreatedAt.Before(query.createdAfter) {
					continue
				}

				switch backup.Status {
				case apiclient.PhysicalBackupStatusCOMPLETED:
					completedBackup = backup

					return nil
				case apiclient.PhysicalBackupStatusERROR, apiclient.PhysicalBackupStatusCANCELED,
					apiclient.PhysicalBackupStatusCHAINBROKEN:
					failMessage := ""
					if backup.FailMessage != nil {
						failMessage = *backup.FailMessage
					}

					return fmt.Errorf("the physical backup %s ended %s: %s", backup.Id, backup.Status, failMessage)
				case apiclient.PhysicalBackupStatusINPROGRESS:
				}
			}

			return errBackupRunning
		})

	return completedBackup
}

func requestPointInTimeToken(
	ctx context.Context,
	owner accountSession,
	chain physicalChain,
) (*apiclient.CreatePhysicalRestoreTokenResponse, error) {
	var request apiclient.PhysicalRestoreTokenCreate

	err := request.FromPhysicalPointInTimeRestoreTokenCreate(apiclient.PhysicalPointInTimeRestoreTokenCreate{
		Type:       apiclient.PhysicalPointInTimeRestoreTokenCreateTypePOINTINTIME,
		DatabaseId: chain.databaseID,
		TargetTime: &chain.targetTime,
	})
	if err != nil {
		return nil, err
	}

	return owner.Client.CreatePhysicalRestoreTokenWithResponse(ctx, request)
}

// requirePointInTimeRestoreOfSnapshot downloads the restore set the recovered catalog names for the
// target time, rebuilds the cluster on the host with the recovery script and checks its markers.
func requirePointInTimeRestoreOfSnapshot(
	t *testing.T,
	cycle *recoveryCycle,
	instance runningInstance,
	owner accountSession,
	chain physicalChain,
) {
	t.Helper()

	tokenResponse, err := requestPointInTimeToken(t.Context(), owner, chain)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, tokenResponse.StatusCode(), string(tokenResponse.Body))

	workDirectory := t.TempDir()
	restoreSetPath := filepath.Join(workDirectory, "restore.tar")
	downloadRestoreSet(t, instance, tokenResponse.JSON201.Token, restoreSetPath)

	scriptResponse, err := owner.Client.GetRecoveryScriptWithResponse(t.Context())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, scriptResponse.StatusCode(), string(scriptResponse.Body))
	scriptPath := filepath.Join(workDirectory, "databasus-recovery.sh")
	require.NoError(t, os.WriteFile(scriptPath, scriptResponse.Body, 0o700))

	outputDirectory := filepath.Join(t.TempDir(), "restored cluster")
	output, err := process.CombinedOutput(t.Context(), process.Spec{Path: "sh", Args: []string{
		scriptPath, "--combine-image", cycle.postgresImage,
		"--target-time", chain.targetTime.UTC().Format("2006-01-02 15:04:05-07:00"),
		restoreSetPath, outputDirectory,
	}})
	require.NoError(t, err, "the recovery script rebuilds the cluster:\n%s", output)

	endpoint := containers.StartPostgresWithBoundDataDir(t, sourceServerImage, filepath.Join(outputDirectory, "data"))

	var restoredPhases []string

	testwait.Within(backupLimit).
		Until(t, "the restored cluster finishes its recovery", func(ctx context.Context) error {
			connection, err := pgx.Connect(ctx, fmt.Sprintf(
				"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=5",
				endpoint.Host, endpoint.Port, sourceUsername, sourcePassword, sourceDatabaseName,
			))
			if err != nil {
				return err
			}
			defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()

			var isInRecovery bool
			if err := connection.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&isInRecovery); err != nil {
				return err
			}

			if isInRecovery {
				return fmt.Errorf("the restored cluster is still in recovery")
			}

			rows, err := connection.Query(ctx, "SELECT phase FROM "+physicalMarkerTable+" ORDER BY phase")
			if err != nil {
				return err
			}

			restoredPhases, err = pgx.CollectRows(rows, pgx.RowTo[string])

			return err
		})

	require.ElementsMatch(t, getMarkersAtTarget(), restoredPhases)
}

func downloadRestoreSet(t *testing.T, instance runningInstance, token, destinationPath string) {
	t.Helper()

	response, err := instance.newClient(t).DownloadPhysicalRestoreSet(t.Context(),
		&apiclient.DownloadPhysicalRestoreSetParams{Token: token})
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)

	destination, err := os.Create(destinationPath)
	require.NoError(t, err)

	_, err = io.Copy(destination, response.Body)
	require.NoError(t, err)
	require.NoError(t, destination.Close())
}
