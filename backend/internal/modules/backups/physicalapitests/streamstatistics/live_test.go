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
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
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
	t.Parallel()

	t.Run("reports current values when the application runs a real WAL streamer", func(t *testing.T) {
		t.Parallel()

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
			code           apiclient.ProblemCode
		}{
			{"returns 404 without values when the user belongs to another workspace", outsider, database.Id, http.StatusNotFound, apiclient.ProblemCodeNotFound},
			{"returns 401 without values when the request has no session", apptest.Session{}, database.Id, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated},
			{"returns 404 without values when the database does not exist", owner, uuid.New(), http.StatusNotFound, apiclient.ProblemCodeNotFound},
			{"returns 400 without values when the database uses logical backups", owner, logicalDatabaseID, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				status, body := testApp.CallPhysicalAPI(
					t,
					testCase.session,
					liveStreamRequest(testCase.databaseID),
				)
				assert.Equal(t, testCase.expectedStatus, status, "%s", body)
				assert.NotContains(t, string(body), "localQueueBytes")

				var problem apiclient.Problem
				require.NoError(t, json.Unmarshal(body, &problem))
				assert.Equal(t, testCase.code, problem.Code)
			})
		}
	})
	t.Run("returns no current values when no streamer runs", func(t *testing.T) {
		t.Parallel()

		testApp := apptest.New(t)
		owner := testApp.CreateAdmin(t)
		workspace := apptest.CreateWorkspace(t, owner, "Idle WAL stream")
		databaseID := testApp.InsertPhysicalDatabase(t, workspace.Id)

		status, body := testApp.CallPhysicalAPI(
			t,
			owner,
			liveStreamRequest(databaseID),
		)

		require.Equal(t, http.StatusOK, status, "%s", body)
		assert.JSONEq(t, `{"isStreamerRunning":false,"current":null}`, string(body))
	})
}

func liveStreamRequest(databaseID uuid.UUID) apptest.PhysicalRequest {
	return apptest.PhysicalLiveRequest(databaseID)
}

func getLiveStream(t *testing.T, testApp *apptest.App, owner apptest.Session, databaseID uuid.UUID) liveStreamReading {
	t.Helper()
	status, body := testApp.CallPhysicalAPI(
		t,
		owner,
		liveStreamRequest(databaseID),
	)
	require.Equal(t, http.StatusOK, status, "%s", body)
	var reading liveStreamReading
	require.NoError(t, json.Unmarshal(body, &reading))
	return reading
}
