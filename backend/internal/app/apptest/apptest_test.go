package apptest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
)

const isolationMarkerVersion = 99990101000000

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("passes the response through the middleware of the server", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		client := apiclient.New(t, testApp.URL)

		response, err := client.GetVersionWithResponse(t.Context())

		require.NoError(t, err)
		assert.NotEmpty(t, response.HTTPResponse.Header.Get("X-Request-Id"))
	})

	t.Run("writes a row into the database of its own test", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)

		db, err := database.Open(t.Context(), testApp.databaseDsn)
		require.NoError(t, err)
		t.Cleanup(func() { closeDatabase(t, db) })

		require.NoError(t, db.WithContext(t.Context()).Exec(
			"INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, TRUE)", isolationMarkerVersion,
		).Error)
	})

	t.Run("does not see the row a previous test wrote when built after it", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)

		db, err := database.Open(t.Context(), testApp.databaseDsn)
		require.NoError(t, err)
		t.Cleanup(func() { closeDatabase(t, db) })

		var markerCount int64
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT COUNT(*) FROM goose_db_version WHERE version_id = ?", isolationMarkerVersion,
		).Scan(&markerCount).Error)
		assert.Zero(t, markerCount)
	})
}

func TestRunJobPass(t *testing.T) {
	t.Parallel()

	t.Run("refuses a WAL manager pass when live physical jobs are disabled", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		recordingTest := &fatalRecorder{TB: t, failure: "", mutex: sync.Mutex{}}
		isDone := make(chan struct{})
		go func() { defer close(isDone); testApp.RunJobPass(recordingTest, "backups_physical_wal_stream_manager") }()
		<-isDone
		assert.Contains(t, recordingTest.getFailure(), "runs only with live physical jobs")
	})

	t.Run("starts a full backup when one scheduler pass finds a database without a full backup", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)

		testApp.RunJobPass(t, "backups_physical_scheduler")

		dispatchedBackups := listPhysicalBackups(t, testApp, owner, database.Id)
		require.Len(t, dispatchedBackups, 1)
		assert.Equal(t, "FULL", dispatchedBackups[0].Type)
		require.Contains(t, []string{"IN_PROGRESS", "COMPLETED"}, dispatchedBackups[0].Status)

		testwait.Until(t, "the dispatched physical backup completes without another pass", func(context.Context) error {
			listedBackups := listPhysicalBackups(t, testApp, owner, database.Id)
			require.Len(t, listedBackups, 1)
			if listedBackups[0].Status != "COMPLETED" {
				return fmt.Errorf("backup status: %s", listedBackups[0].Status)
			}
			return nil
		})
	})

	t.Run("starts a requested full backup when one pass of the backup request job runs", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)
		requestFullBackup(t, testApp, owner, database.Id)

		testApp.RunJobPass(t, "backups_physical_request_scheduler")

		dispatchedBackups := listPhysicalBackups(t, testApp, owner, database.Id)
		require.Len(t, dispatchedBackups, 1)
		assert.Equal(t, "FULL", dispatchedBackups[0].Type)
		assert.Contains(t, []string{"IN_PROGRESS", "COMPLETED"}, dispatchedBackups[0].Status)
	})

	t.Run("starts a requested full backup when one scheduler pass runs", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)
		requestFullBackup(t, testApp, owner, database.Id)

		testApp.RunJobPass(t, "backups_physical_scheduler")

		dispatchedBackups := listPhysicalBackups(t, testApp, owner, database.Id)
		require.Len(t, dispatchedBackups, 1)
		assert.Equal(t, "FULL", dispatchedBackups[0].Type)
		assert.Contains(t, []string{"IN_PROGRESS", "COMPLETED"}, dispatchedBackups[0].Status)
	})

	t.Run("fails the test when no job has the name", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		recordingTest := &fatalRecorder{TB: t, failure: "", mutex: sync.Mutex{}}

		isFinished := make(chan struct{})
		go func() {
			defer close(isFinished)
			testApp.RunJobPass(recordingTest, "no_such_job")
		}()
		<-isFinished

		assert.Contains(t, recordingTest.getFailure(), "no_such_job")
	})
}

func closeDatabase(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

// fatalRecorder stops the goroutine that called Fatalf, as testing.T does, and keeps the message
// for the surrounding test instead of failing it.
type fatalRecorder struct {
	testing.TB

	failure string
	mutex   sync.Mutex
}

func (r *fatalRecorder) Fatalf(format string, args ...any) {
	r.mutex.Lock()
	r.failure = fmt.Sprintf(format, args...)
	r.mutex.Unlock()

	runtime.Goexit()
}

func (r *fatalRecorder) Errorf(format string, args ...any) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = fmt.Sprintf(format, args...)
}

func (r *fatalRecorder) getFailure() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.failure
}

func TestPhysicalBackupLoops(t *testing.T) {
	t.Parallel()

	t.Run("records distinct sample times and exposes no manual clock with live jobs", func(t *testing.T) {
		t.Parallel()

		testApp := New(t, WithPhysicalBackupLoops())
		require.Nil(t, testApp.Clock)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, backupDatabase := enablePhysicalBackupLoops(t, testApp, source)
		var firstSample time.Time
		testwait.Until(t, "the WAL streamer records two samples at distinct times", func(context.Context) error {
			statistics := getPhysicalStreamStatistics(t, testApp, owner, backupDatabase.Id)
			if statistics.Streamer.State != "RUNNING" || statistics.Current == nil {
				return fmt.Errorf("no running sampled WAL session")
			}
			if firstSample.IsZero() {
				firstSample = statistics.Current.SampledAt
			}
			if !statistics.Current.SampledAt.After(firstSample) {
				return fmt.Errorf("waiting for the second WAL sample")
			}
			return nil
		})
	})
	t.Run("keeps orphan WAL until an explicit cleaner pass with live jobs", func(t *testing.T) {
		t.Parallel()

		testApp := New(t, WithPhysicalBackupLoops())
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, backupDatabase := enablePhysicalBackupLoops(t, testApp, source)
		testApp.WaitForPhysicalBackups(t)

		backupConfig := GetBackupConfig(t, owner, backupDatabase.Id)
		require.NotNil(t, backupConfig.StorageId)
		segment := testApp.InsertPhysicalWalSegment(
			t,
			PhysicalWalSegmentSeed{
				DatabaseID:  backupDatabase.Id,
				StorageID:   *backupConfig.StorageId,
				TimelineID:  999,
				FileName:    new("orphan-live-wal"),
				WalFilename: "000003E70000000000000001",
				StartLSN:    "0/1000000",
				EndLSN:      "0/2000000",
				ReceivedAt:  time.Now().Add(-2 * time.Hour),
			},
		)

		observationEnds := time.Now().Add(time.Minute + time.Second)
		testwait.Until(t, "a cleaner interval passes without deleting orphan WAL", func(context.Context) error {
			var segmentCount int
			testApp.QuerySQL(
				t,
				&segmentCount,
				"SELECT count(*) FROM backups_physical_wal_segments WHERE id=?",
				segment.ID,
			)
			require.Equal(t, 1, segmentCount)
			if time.Now().Before(observationEnds) {
				return fmt.Errorf("the cleaner observation window is still open")
			}
			return nil
		})
		testApp.RunJobPass(t, "backups_physical_cleaner")

		var segmentCount int
		testApp.QuerySQL(t, &segmentCount, "SELECT count(*) FROM backups_physical_wal_segments WHERE id=?", segment.ID)
		assert.Zero(t, segmentCount)
	})

	t.Run("runs the backup request job live and again after modules restart", func(t *testing.T) {
		t.Parallel()

		testApp := New(t, WithPhysicalBackupLoops())
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)
		testApp.WaitForPhysicalBackups(t)
		waitForCompletedBackupRequestPass(t, testApp)

		testApp.RestartModules(t)
		waitForCompletedBackupRequestPass(t, testApp)

		listedBefore := len(listPhysicalBackups(t, testApp, owner, database.Id))
		requestFullBackup(t, testApp, owner, database.Id)

		testwait.Until(t, "the requested full appears without an explicit pass", func(context.Context) error {
			if listedCount := len(listPhysicalBackups(t, testApp, owner, database.Id)); listedCount <= listedBefore {
				return fmt.Errorf("%d backups are listed", listedCount)
			}

			return nil
		})
	})

	t.Run("runs one pass of the backup request job by name with live jobs", func(t *testing.T) {
		t.Parallel()

		testApp := New(t, WithPhysicalBackupLoops())

		testApp.RunJobPass(t, "backups_physical_request_scheduler")

		var requestJobSnapshot background.JobSnapshot
		for _, snapshot := range testApp.GetPhysicalLoopSnapshots() {
			if snapshot.Name == "backups_physical_request_scheduler" {
				requestJobSnapshot = snapshot
			}
		}

		assert.NotNil(t, requestJobSnapshot.LastCompletedAt)
	})

	t.Run("leaves a requested backup unstarted without the option and without a pass", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)
		requestFullBackup(t, testApp, owner, database.Id)

		// The backup request job would run every second; observe five of its intervals.
		observationEnds := time.Now().Add(5 * time.Second)
		testwait.Until(t, "five intervals pass without a started backup", func(context.Context) error {
			require.Empty(t, listPhysicalBackups(t, testApp, owner, database.Id))

			if time.Now().Before(observationEnds) {
				return fmt.Errorf("observation window is still open")
			}

			return nil
		})

		var pendingRequests int64
		testApp.QuerySQL(
			t,
			&pendingRequests,
			"SELECT count(*) FROM backups_configs_physical WHERE database_id=? AND force_full_requested_at IS NOT NULL",
			database.Id,
		)
		assert.Equal(t, int64(1), pendingRequests)
	})

	t.Run("starts no physical backup or wal stream without the option", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)

		// The backup request job would run every second; observe five of its intervals after the config save answered.
		observationEnds := time.Now().Add(5 * time.Second)
		testwait.Until(t, "five intervals pass without physical work", func(context.Context) error {
			status, body := testApp.CallPhysicalAPI(t, owner, PhysicalListRequest(database.Id, "?limit=1000"))
			require.Equal(t, http.StatusOK, status, "%s", body)
			var listed struct {
				Backups []json.RawMessage `json:"items"`
			}
			require.NoError(t, json.Unmarshal(body, &listed))
			require.Empty(t, listed.Backups)

			var slotCount int
			conn, err := pgx.Connect(
				t.Context(),
				fmt.Sprintf(
					"host=%s port=%d user=testuser password=testpassword dbname=testdb sslmode=disable",
					source.Host,
					source.Port,
				),
			)
			require.NoError(t, err)
			require.NoError(
				t,
				conn.QueryRow(t.Context(), `SELECT COUNT(*) FROM pg_replication_slots WHERE slot_name LIKE 'databasus_slot_%'`).
					Scan(&slotCount),
			)
			require.NoError(t, conn.Close(t.Context()))
			require.Zero(t, slotCount)
			if time.Now().Before(observationEnds) {
				return fmt.Errorf("observation window is still open")
			}
			return nil
		})
	})
	t.Run("stops physical work before closing the database when an application ends", func(t *testing.T) {
		t.Parallel()

		source := containers.StartPhysicalPostgres(t, "postgres:17")
		for applicationNumber := range 2 {
			t.Run(fmt.Sprintf("application %d", applicationNumber+1), func(t *testing.T) {
				testApp := New(t, WithPhysicalBackupLoops())
				owner, createdDatabase := enablePhysicalBackupLoops(t, testApp, source)
				testwait.Until(t, "the application owns a running WAL stream", func(context.Context) error {
					status, body := testApp.CallPhysicalAPI(
						t,
						owner,
						PhysicalStatisticsRequest(createdDatabase.Id, "HOUR"),
					)
					require.Equal(t, http.StatusOK, status, "%s", body)
					var statistics struct {
						Streamer struct {
							State string `json:"state"`
						} `json:"streamer"`
					}
					require.NoError(t, json.Unmarshal(body, &statistics))
					if statistics.Streamer.State != "RUNNING" {
						return fmt.Errorf("streamer state: %s", statistics.Streamer.State)
					}
					if getActivePhysicalReplicationSlotCount(t, source) != 1 {
						return fmt.Errorf("the source has no active WAL replication slot")
					}
					return nil
				})
			})
			assert.Zero(
				t,
				getActivePhysicalReplicationSlotCount(t, source),
				"no receiver remains after the application closes",
			)
		}
	})

	t.Run("reports a cleanup failure when a background job outlives the stop bound", func(t *testing.T) {
		t.Parallel()

		var recordingTest *fatalRecorder
		t.Run("application cleanup", func(t *testing.T) {
			recordingTest = &fatalRecorder{TB: t, failure: "", mutex: sync.Mutex{}}
			releaseJob := make(chan struct{})
			startedJob := make(chan struct{})
			t.Cleanup(func() { close(releaseJob) })

			testApp := New(
				recordingTest,
				func(applied *settings) { applied.physicalBackupStopTimeout = 100 * time.Millisecond },
			)
			testApp.physicalRunner = background.NewRunner(
				background.RunnerDeps{Logger: slog.Default(), Clock: clock.NewSystem()},
				background.Job{
					Name:     "held_job",
					Interval: time.Hour,
					Run: func(context.Context, time.Time) error {
						close(startedJob)
						<-releaseJob
						return nil
					},
				},
			)

			loopsCtx, cancel := context.WithCancel(t.Context())
			testApp.cancelPhysicalLoops = cancel
			testApp.physicalRunner.Start(loopsCtx)
			select {
			case <-startedJob:
			case <-t.Context().Done():
				t.Fatal("held job did not start")
			}
		})
		assert.Contains(t, recordingTest.getFailure(), "stop physical backup loops and runs")
		assert.Contains(t, recordingTest.getFailure(), context.DeadlineExceeded.Error())
	})
}

func enablePhysicalBackupLoops(t *testing.T, testApp *App, source containers.Endpoint) (Session, apiclient.Database) {
	t.Helper()
	owner := testApp.CreateAdmin(t)
	workspace := CreateWorkspace(t, owner, "Physical backups")
	storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)
	password := "testpassword"

	var settings apiclient.DatabaseSettings
	require.NoError(t, settings.FromPostgresqlPhysicalSettings(apiclient.PostgresqlPhysicalSettings{
		Type: apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
		Host: source.Host, Port: source.Port, Username: "testuser", Password: &password,
		SslMode: apiclient.PostgresqlSslModeDisable, SshTunnel: apiclient.SshTunnelSettings{IsEnabled: false},
		BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
	}))

	database := CreateDatabase(t, owner, workspace.Id, settings)
	backupConfig := GetBackupConfig(t, owner, database.Id)
	response, err := owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &storageID,
			Encryption:       apiclient.BackupEncryptionNONE,
			Settings:         backupConfig.Settings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	return owner, database
}

func requestFullBackup(t *testing.T, testApp *App, owner Session, databaseID uuid.UUID) {
	t.Helper()
	status, body := testApp.CallPhysicalAPI(t, owner, PhysicalStartRequest(databaseID, "FULL"))
	require.Equal(t, http.StatusAccepted, status, "%s", body)
}

func waitForCompletedBackupRequestPass(t *testing.T, testApp *App) {
	t.Helper()
	testwait.Until(t, "the live backup request job completes a pass", func(context.Context) error {
		for _, snapshot := range testApp.GetPhysicalLoopSnapshots() {
			if snapshot.Name == "backups_physical_request_scheduler" && snapshot.LastCompletedAt != nil {
				return nil
			}
		}

		return fmt.Errorf("the live runner has no completed pass of the backup request job")
	})
}

func getActivePhysicalReplicationSlotCount(t *testing.T, source containers.Endpoint) int {
	t.Helper()
	conn, err := pgx.Connect(
		t.Context(),
		fmt.Sprintf(
			"host=%s port=%d user=testuser password=testpassword dbname=testdb sslmode=disable",
			source.Host,
			source.Port,
		),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close(context.WithoutCancel(t.Context()))) }()

	var activeSlotCount int
	require.NoError(
		t,
		conn.QueryRow(t.Context(), "SELECT COUNT(*) FROM pg_replication_slots WHERE active AND slot_name LIKE 'databasus_slot_%'").
			Scan(&activeSlotCount),
	)
	return activeSlotCount
}

type physicalBackupRow struct {
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	FailMessage   *string   `json:"failMessage"`
	ToWalFilename *string   `json:"toWalFilename"`
}

func listPhysicalBackups(t *testing.T, testApp *App, owner Session, databaseID uuid.UUID) []physicalBackupRow {
	t.Helper()
	status, body := testApp.CallPhysicalAPI(t, owner, PhysicalListRequest(databaseID, "?limit=1000"))
	require.Equal(t, http.StatusOK, status, "%s", body)
	var listed struct {
		Backups []physicalBackupRow `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &listed))
	return listed.Backups
}

func TestPhysicalStartupWork(t *testing.T) {
	t.Parallel()

	t.Run("fails an unfinished backup when modules restart", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)
		config := GetBackupConfig(t, owner, database.Id)
		require.NotNil(t, config.StorageId)
		storageID := *config.StorageId
		backup := testApp.InsertPhysicalFullBackup(t, PhysicalFullBackupSeed{
			DatabaseID: database.Id, StorageID: storageID, Status: "IN_PROGRESS",
		})
		backupID := backup.ID

		testApp.RestartModules(t)

		listedBackups := listPhysicalBackups(t, testApp, owner, database.Id)
		require.Len(t, listedBackups, 1)
		assert.Equal(t, backupID, listedBackups[0].ID)
		assert.Equal(t, "ERROR", listedBackups[0].Status)
		require.NotNil(t, listedBackups[0].FailMessage)
		assert.Contains(t, *listedBackups[0].FailMessage, "application restart")
	})
	t.Run("removes an orphan WAL segment when one cleaner pass runs", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)
		testApp.ExecuteSQL(t, `INSERT INTO backups_physical_wal_segments
   (database_id, storage_id, timeline_id, wal_filename, start_lsn, end_lsn, file_name, received_at, claimed_at)
   SELECT database_id, storage_id, 1, '000000010000000000000001', '0/1000000', '0/2000000',
    'orphan-wal', ?, ? FROM backups_configs WHERE database_id = ?`,
			time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour), database.Id)
		require.Len(t, listPhysicalBackups(t, testApp, owner, database.Id), 1)

		testApp.RunJobPass(t, "backups_physical_cleaner")

		assert.Empty(t, listPhysicalBackups(t, testApp, owner, database.Id))
	})
	t.Run("removes an orphan backup slot and fails a stale streamer when modules restart", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		_, database := enablePhysicalBackupLoops(t, testApp, source)
		conn := openPhysicalSource(t, source)
		var backupSlotName string
		testApp.QuerySQL(t, &backupSlotName,
			"SELECT backup_slot_name FROM databases_postgresql_physical WHERE database_id = ?", database.Id)
		_, err := conn.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1)", backupSlotName)
		require.NoError(t, err)
		testApp.InsertPhysicalWalStreamer(t, PhysicalWalStreamerSeed{
			DatabaseID: database.Id, LastHeartbeatAt: time.Now().Add(-5 * time.Minute),
		})

		testApp.RestartModules(t)

		var slotCount int
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT count(*) FROM pg_replication_slots WHERE slot_name = $1", backupSlotName).Scan(&slotCount))
		assert.Zero(t, slotCount)
		var streamerStatus string
		testApp.QuerySQL(t, &streamerStatus,
			"SELECT status FROM backups_physical_wal_streamers WHERE database_id = ?", database.Id)
		assert.Equal(t, "FAILED", streamerStatus)
	})
	t.Run("keeps the replication slot and starts a new session when live modules restart", func(t *testing.T) {
		t.Parallel()

		testApp := New(t, WithPhysicalBackupLoops())
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)
		var initialSession time.Time
		var initialRestarts int
		testwait.Until(t, "the first WAL session records a sample", func(context.Context) error {
			statistics := getPhysicalStreamStatistics(t, testApp, owner, database.Id)
			if statistics.Streamer.State != "RUNNING" || statistics.Current == nil ||
				getActivePhysicalReplicationSlotCount(t, source) != 1 {
				return fmt.Errorf("no running sampled session")
			}
			initialSession = statistics.Current.SessionStartedAt
			initialRestarts = statistics.Streamer.RestartsLastHour
			return nil
		})
		testApp.WaitForPhysicalBackups(t)
		testApp.RestartModules(t)

		conn := openPhysicalSource(t, source)
		var slotCount int
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT count(*) FROM pg_replication_slots WHERE slot_name LIKE 'databasus_slot_%'").Scan(&slotCount))
		assert.Equal(t, 1, slotCount)
		testwait.Until(t, "the restarted WAL session records a sample", func(context.Context) error {
			statistics := getPhysicalStreamStatistics(t, testApp, owner, database.Id)
			if statistics.Current == nil || !statistics.Current.SessionStartedAt.After(initialSession) {
				return fmt.Errorf("no sample from a new session")
			}
			assert.Equal(t, initialRestarts+1, statistics.Streamer.RestartsLastHour)
			return nil
		})
	})
}

type physicalStreamStatistics struct {
	Streamer struct {
		State            string `json:"state"`
		RestartsLastHour int    `json:"restartsLastHour"`
	} `json:"streamer"`
	Current *struct {
		SessionStartedAt time.Time `json:"sessionStartedAt"`
		SampledAt        time.Time `json:"sampledAt"`
	} `json:"current"`
}

func getPhysicalStreamStatistics(
	t *testing.T,
	testApp *App,
	owner Session,
	databaseID uuid.UUID,
) physicalStreamStatistics {
	t.Helper()
	status, body := testApp.CallPhysicalAPI(t, owner, PhysicalStatisticsRequest(databaseID, "HOUR"))
	require.Equal(t, http.StatusOK, status, "%s", body)
	var statistics physicalStreamStatistics
	require.NoError(t, json.Unmarshal(body, &statistics))
	return statistics
}

func openPhysicalSource(t *testing.T, source containers.Endpoint) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), fmt.Sprintf(
		"host=%s port=%d user=testuser password=testpassword dbname=testdb sslmode=disable", source.Host, source.Port))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close(context.WithoutCancel(t.Context()))) })
	return conn
}

func TestWalStreamTimings(t *testing.T) {
	t.Parallel()

	t.Run("archives WAL from a quiet source when the application shortens forced rotation", func(t *testing.T) {
		t.Parallel()

		timings := backups.ProductionWalStreamTimings()
		timings.ForcedRotationInterval = 2 * time.Second
		testApp := New(t, WithPhysicalBackupLoops(), WithWalStreamTimings(timings))
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)

		testwait.Until(t, "the WAL receiver owns the source slot", func(context.Context) error {
			if getActivePhysicalReplicationSlotCount(t, source) != 1 {
				return fmt.Errorf("no active WAL receiver")
			}
			return nil
		})
		testwait.Until(
			t,
			"the initial full backup completes before measuring forced rotation",
			func(context.Context) error {
				for _, backup := range listPhysicalBackups(t, testApp, owner, database.Id) {
					if backup.Type == "FULL" && backup.Status == "COMPLETED" {
						return nil
					}
				}
				return fmt.Errorf("the initial full backup has not completed")
			},
		)

		conn := openPhysicalSource(t, source)
		var quietWalFilename string
		require.NoError(
			t,
			conn.QueryRow(t.Context(), "SELECT pg_walfile_name(pg_current_wal_lsn())").Scan(&quietWalFilename),
		)
		_, err := conn.Exec(t.Context(), "CREATE TABLE quiet_source (id int)")
		require.NoError(t, err)

		rotationStartedAt := time.Now()
		rotationTimeout := 10 * timings.ForcedRotationInterval
		testwait.Until(t, "forced rotation archives a segment from the quiet source", func(context.Context) error {
			listedBackups := listPhysicalBackups(t, testApp, owner, database.Id)
			require.Less(
				t,
				time.Since(rotationStartedAt),
				rotationTimeout,
				"forced rotation did not archive the quiet source WAL in time",
			)
			for _, backup := range listedBackups {
				if backup.Type == "WAL" && backup.ToWalFilename != nil && *backup.ToWalFilename >= quietWalFilename {
					return nil
				}
			}
			return fmt.Errorf("no archived WAL segment")
		})
		assert.Equal(t, 5*time.Minute, backups.ProductionWalStreamTimings().ForcedRotationInterval)
	})
}
