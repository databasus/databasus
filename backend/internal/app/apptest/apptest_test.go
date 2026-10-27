package apptest

import (
	"context"
	"encoding/json"
	"fmt"
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

	backups_bridge "databasus-backend/internal/features/backups/config"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/util/testing/containers"
)

const isolationMarkerVersion = 99990101000000

func TestNew(t *testing.T) {
	t.Run("passes the response through the middleware of the server", func(t *testing.T) {
		testApp := New(t)
		client := apiclient.New(t, testApp.URL)

		response, err := client.GetVersionWithResponse(t.Context())

		require.NoError(t, err)
		assert.NotEmpty(t, response.HTTPResponse.Header.Get("X-Request-Id"))
	})

	t.Run("writes a row into the database of its own test", func(t *testing.T) {
		testApp := New(t)

		db, err := database.Open(t.Context(), testApp.databaseDsn)
		require.NoError(t, err)
		t.Cleanup(func() { closeDatabase(t, db) })

		require.NoError(t, db.WithContext(t.Context()).Exec(
			"INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, TRUE)", isolationMarkerVersion,
		).Error)
	})

	t.Run("does not see the row a previous test wrote when built after it", func(t *testing.T) {
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

	t.Run("fails the test when another test application is alive", func(t *testing.T) {
		New(t)
		secondTest := &fatalRecorder{TB: t, failure: "", mutex: sync.Mutex{}}

		isFinished := make(chan struct{})
		go func() {
			defer close(isFinished)
			New(secondTest)
		}()
		<-isFinished

		assert.Contains(t, secondTest.getFailure(), "run one at a time")
	})
}

func TestRunJobPass(t *testing.T) {
	t.Run("fails the test when no job has the name", func(t *testing.T) {
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
	t.Run("starts no physical backup or wal stream without the option", func(t *testing.T) {
		testApp := New(t)
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)

		// The scheduler ticks every second; observe five ticks after the config save answered.
		observationEnds := time.Now().Add(5 * time.Second)
		testwait.Until(t, "five scheduler ticks pass without physical work", func(context.Context) error {
			status, body := testApp.CallOldAPI(t, owner, OldAPICall{
				Method: http.MethodGet,
				Path:   "/backups/physical/database/" + database.Id.String() + "/backups?limit=1000",
			})
			require.Equal(t, http.StatusOK, status, "%s", body)
			var listed struct {
				Backups []json.RawMessage `json:"backups"`
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
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		for applicationNumber := range 2 {
			t.Run(fmt.Sprintf("application %d", applicationNumber+1), func(t *testing.T) {
				testApp := New(t, WithPhysicalBackupLoops())
				owner, createdDatabase := enablePhysicalBackupLoops(t, testApp, source)
				testwait.Until(t, "the application owns a running WAL stream", func(context.Context) error {
					status, body := testApp.CallOldAPI(t, owner, OldAPICall{
						Method: http.MethodGet,
						Path:   "/backups/physical/database/" + createdDatabase.Id.String() + "/wal-stream/statistics?period=HOUR",
					})
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
			stopCtx, cancelStop := context.WithTimeout(t.Context(), time.Second)
			require.NoError(t, backups_bridge.GetCancellationRegistry().WaitUntilEmpty(stopCtx))
			cancelStop()
			assert.Zero(
				t,
				getActivePhysicalReplicationSlotCount(t, source),
				"no receiver remains after the application closes",
			)
		}
	})

	t.Run("reports a cleanup failure when registered work outlives the stop bound", func(t *testing.T) {
		var recordingTest *fatalRecorder
		t.Run("application cleanup", func(t *testing.T) {
			recordingTest = &fatalRecorder{TB: t, failure: "", mutex: sync.Mutex{}}
			registry := backups_bridge.GetCancellationRegistry()
			runID := uuid.New()
			t.Cleanup(func() { registry.Unregister(runID) })
			New(recordingTest, WithPhysicalBackupLoops(), func(applied *settings) {
				applied.physicalBackupStopTimeout = 100 * time.Millisecond
			})
			registry.Register(runID, func() {})
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
