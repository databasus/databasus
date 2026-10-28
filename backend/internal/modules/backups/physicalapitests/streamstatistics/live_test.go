package streamstatistics_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/util/testing/containers"
)

type liveStreamReading struct {
	IsStreamerRunning bool `json:"isStreamerRunning"`
	Current           *struct {
		MeasuredAt           time.Time  `json:"measuredAt"`
		SourceProbedAt       *time.Time `json:"sourceProbedAt"`
		SourceState          *string    `json:"sourceState"`
		SourceCurrentLSN     *string    `json:"sourceCurrentLsn"`
		SlotLagBytes         *int64     `json:"slotLagBytes"`
		LocalQueueBytes      int64      `json:"localQueueBytes"`
		LocalQueuePauseBytes int64      `json:"localQueuePauseBytes"`
	} `json:"current"`
}

func TestGetWalStreamLive(t *testing.T) {
	t.Run("reports current values when the application runs a real WAL streamer", func(t *testing.T) {
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		timings := backups.ProductionWalStreamTimings()
		timings.ForcedRotationInterval = 2 * time.Second
		testApp := apptest.New(t, apptest.WithPhysicalBackupLoops(), apptest.WithWalStreamTimings(timings))
		owner := testApp.CreateAdmin(t)
		workspace := apptest.CreateWorkspace(t, owner, "Live WAL stream")
		storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)
		password := "testpassword"
		var settings apiclient.DatabaseSettings
		require.NoError(t, settings.FromPostgresqlPhysicalSettings(apiclient.PostgresqlPhysicalSettings{
			Type: apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
			Host: source.Host, Port: source.Port, Username: "testuser", Password: &password,
			SslMode:    apiclient.PostgresqlSslModeDisable,
			SshTunnel:  apiclient.SshTunnelSettings{IsEnabled: false},
			BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
		}))
		database := apptest.CreateDatabase(t, owner, workspace.Id, settings)
		apptest.SaveBackupConfigWithStorage(t, owner, apptest.StorageUse{DatabaseID: database.Id, StorageID: storageID})

		var firstReading liveStreamReading
		testwait.Until(
			t,
			"the running streamer reports a source probe through the live route",
			func(context.Context) error {
				firstReading = getLiveStream(t, testApp, owner, database.Id)
				if !firstReading.IsStreamerRunning || firstReading.Current == nil ||
					firstReading.Current.SourceProbedAt == nil {
					return fmt.Errorf("no running streamer with a source probe")
				}
				return nil
			},
		)

		assert.WithinDuration(t, time.Now().UTC(), firstReading.Current.MeasuredAt, 5*time.Second)
		require.NotNil(t, firstReading.Current.SourceState)
		assert.Equal(t, "REACHABLE", *firstReading.Current.SourceState)
		assert.NotNil(t, firstReading.Current.SourceCurrentLSN)
		assert.NotNil(t, firstReading.Current.SlotLagBytes)
		assert.GreaterOrEqual(t, firstReading.Current.LocalQueueBytes, int64(0))
		assert.Positive(t, firstReading.Current.LocalQueuePauseBytes)
		secondReading := getLiveStream(t, testApp, owner, database.Id)
		require.NotNil(t, secondReading.Current)
		assert.True(t, secondReading.Current.MeasuredAt.After(firstReading.Current.MeasuredAt))

		outsider := testApp.CreateNonAdminUser(t, owner)
		logicalDatabaseID := testApp.InsertDatabase(t, workspace.Id)
		for _, testCase := range []struct {
			name           string
			session        apptest.Session
			databaseID     uuid.UUID
			expectedStatus int
		}{
			{"returns 400 without values when the user belongs to another workspace", outsider, database.Id, http.StatusBadRequest},
			{"returns 401 without values when the request has no session", apptest.Session{}, database.Id, http.StatusUnauthorized},
			{"returns 400 without values when the database uses logical backups", owner, logicalDatabaseID, http.StatusBadRequest},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				status, body := testApp.CallOldAPI(
					t,
					testCase.session,
					apptest.OldAPICall{Method: http.MethodGet, Path: liveStreamPath(testCase.databaseID)},
				)
				assert.Equal(t, testCase.expectedStatus, status, "%s", body)
				assert.NotContains(t, string(body), "localQueueBytes")
			})
		}
	})
	t.Run("returns no current values when no streamer runs", func(t *testing.T) {
		testApp := apptest.New(t)
		owner := testApp.CreateAdmin(t)
		workspace := apptest.CreateWorkspace(t, owner, "Idle WAL stream")
		databaseID := testApp.InsertPhysicalDatabase(t, workspace.Id)

		status, body := testApp.CallOldAPI(
			t,
			owner,
			apptest.OldAPICall{Method: http.MethodGet, Path: liveStreamPath(databaseID)},
		)

		require.Equal(t, http.StatusOK, status, "%s", body)
		assert.JSONEq(t, `{"isStreamerRunning":false,"current":null}`, string(body))
	})
}

func liveStreamPath(databaseID uuid.UUID) string {
	return "/backups/physical/database/" + databaseID.String() + "/wal-stream/live"
}

func getLiveStream(t *testing.T, testApp *apptest.App, owner apptest.Session, databaseID uuid.UUID) liveStreamReading {
	t.Helper()
	status, body := testApp.CallOldAPI(
		t,
		owner,
		apptest.OldAPICall{Method: http.MethodGet, Path: liveStreamPath(databaseID)},
	)
	require.Equal(t, http.StatusOK, status, "%s", body)
	var reading liveStreamReading
	require.NoError(t, json.Unmarshal(body, &reading))
	return reading
}
