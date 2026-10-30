package walstreaming_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestClaimPhysicalWalStreamer(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"FAILED", "STALE", "RUNNING"} {
		t.Run("claims only dead ownership when the catalog starts with a "+status+" streamer", func(t *testing.T) {
			team := newWalTeam(t, walTeamSpec{})
			started := time.Now().UTC().Add(-5 * time.Minute)
			heartbeat := started
			rowStatus := status
			if status == "STALE" {
				rowStatus = "RUNNING"
			}
			if status == "RUNNING" {
				heartbeat = time.Now().UTC()
			}
			team.App.InsertPhysicalWalStreamer(
				t,
				apptest.PhysicalWalStreamerSeed{
					DatabaseID:      team.Database.Id,
					Status:          rowStatus,
					StartedAt:       started,
					LastHeartbeatAt: heartbeat,
				},
			)
			team.enable(t)
			if status == "RUNNING" {
				nonStreaming := team.addDatabase(t)
				settings, err := nonStreaming.Database.Settings.AsPostgresqlPhysicalSettings()
				require.NoError(t, err)
				settings.Password = new("testpassword")
				settings.BackupType = apiclient.PhysicalBackupTypeFULLINCREMENTAL
				var databaseSettings apiclient.DatabaseSettings
				require.NoError(t, databaseSettings.FromPostgresqlPhysicalSettings(settings))
				response, err := team.Owner.Client.UpdateDatabaseWithResponse(
					t.Context(),
					nonStreaming.Database.Id,
					apiclient.DatabaseUpdate{Name: nonStreaming.Database.Name, Settings: databaseSettings},
				)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode())
				nonStreaming.enable(t)
				other := team.addDatabase(t)
				other.enable(t)
				other.waitForReceiver(t)
				testwait.Until(
					t,
					"the other streamer records a sample after the supervisor has reconciled",
					func(context.Context) error {
						if other.statistics(t).Current == nil {
							return fmt.Errorf("no sample on the eligible database")
						}
						return nil
					},
				)
				assert.False(t, nonStreaming.live(t).IsStreamerRunning)
				assert.Nil(t, nonStreaming.statistics(t).Current)
				assert.False(t, team.live(t).IsStreamerRunning)
				var persisted time.Time
				team.App.QuerySQL(
					t,
					&persisted,
					"SELECT started_at FROM backups_physical_wal_streamers WHERE database_id=?",
					team.Database.Id,
				)
				assert.WithinDuration(t, started, persisted, time.Microsecond)
				return
			}
			team.waitForReceiver(t)
			var persisted []struct {
				Status    string
				StartedAt time.Time
			}
			team.App.QuerySQL(
				t,
				&persisted,
				"SELECT status,started_at FROM backups_physical_wal_streamers WHERE database_id=?",
				team.Database.Id,
			)
			require.Len(t, persisted, 1)
			assert.Equal(t, "RUNNING", persisted[0].Status)
			assert.True(t, persisted[0].StartedAt.After(started))
		})
	}
}

func TestRecoverStalePhysicalWalStreamer(t *testing.T) {
	t.Parallel()

	t.Run("marks a stale running row failed during startup when the configuration is disabled", func(t *testing.T) {
		team := newWalTeam(t, walTeamSpec{IsWithoutLoops: true})
		team.App.InsertPhysicalWalStreamer(
			t,
			apptest.PhysicalWalStreamerSeed{
				DatabaseID:      team.Database.Id,
				Status:          "RUNNING",
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC().Add(-time.Hour),
			},
		)
		team.App.RestartModules(t)
		var status string
		team.App.QuerySQL(
			t,
			&status,
			"SELECT status FROM backups_physical_wal_streamers WHERE database_id=?",
			team.Database.Id,
		)
		assert.Equal(t, "FAILED", status)
		assert.False(t, team.live(t).IsStreamerRunning)
	})
}
