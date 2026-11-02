package apptest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	"databasus-backend/internal/lib/agentclient"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestTelemetryReport(t *testing.T) {
	t.Parallel()

	t.Run("joins backups by database and selects completed full backups deterministically", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusOK}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)

		team := enginetest.NewTeam(
			t,
			apptest.WithTelemetry(
				apptest.TelemetrySettings{
					ReceiverURL:  server.URL,
					IdentityPath: filepath.Join(t.TempDir(), "identity.json"),
				},
			),
		)
		team.App.Clock.Advance(team.App.Clock.Now().Truncate(time.Microsecond).Sub(team.App.Clock.Now()))

		source := containers.StartPhysicalPostgres(t, "postgres:18")
		logical := team.CreateDatabase(t, apiclient.PostgresqlLogicalSettings{
			Host:      source.Host,
			Port:      source.Port,
			Username:  containers.PostgresUsername,
			Password:  new(containers.PostgresPassword),
			Database:  new(containers.PostgresDatabase),
			SslMode:   apiclient.PostgresqlSslModeDisable,
			SshTunnel: enginetest.NewDisabledSSHTunnelSettings(),
			CpuCount:  1,
		})
		secondLogical := team.CreateDatabase(t, apiclient.PostgresqlLogicalSettings{
			Host:      source.Host,
			Port:      source.Port,
			Username:  containers.PostgresUsername,
			Password:  new(containers.PostgresPassword),
			Database:  new(containers.PostgresDatabase),
			SslMode:   apiclient.PostgresqlSslModeDisable,
			SshTunnel: enginetest.NewDisabledSSHTunnelSettings(),
			CpuCount:  1,
		})
		physical := team.CreateDatabase(t, apiclient.PostgresqlPhysicalSettings{
			Host:       source.Host,
			Port:       source.Port,
			Username:   containers.PostgresUsername,
			Password:   new(containers.PostgresPassword),
			SslMode:    apiclient.PostgresqlSslModeDisable,
			SshTunnel:  enginetest.NewDisabledSSHTunnelSettings(),
			BackupType: apiclient.PhysicalBackupTypeFULL,
		})

		for _, databaseID := range []uuid.UUID{logical.Id, secondLogical.Id, physical.Id} {
			team.App.ExecuteSQL(t, "UPDATE databases SET health_status = 'AVAILABLE' WHERE id = ?", databaseID)
		}
		logicalBackup := team.BackUp(t, logical.Id, apiclient.BackupEncryptionNONE)
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_logical SET backup_size_mb = 0.1, backup_raw_db_size_mb = 2.1 WHERE id = ?",
			logicalBackup.Id,
		)
		team.App.ExecuteSQL(t, "UPDATE databases_postgresql_logical SET port = 1 WHERE database_id = ?", logical.Id)
		failedBackup := team.WaitForBackupOutcome(t, team.StartBackup(t, logical.Id).Id)
		require.Equal(t, apiclient.LogicalBackupStatusFAILED, failedBackup.Status)

		physicalTime := team.App.Clock.Now()
		firstFull := team.App.InsertPhysicalFullBackup(
			t,
			apptest.PhysicalFullBackupSeed{
				DatabaseID: physical.Id,
				StorageID:  team.StorageID,
				CreatedAt:  physicalTime,
				StartLSN:   new("0/1000000"), StopLSN: new("0/2000000"),
				BackupSizeMb: new(5.1),
				RawSizeMb:    new(8.1),
			},
		)
		secondFull := team.App.InsertPhysicalFullBackup(
			t,
			apptest.PhysicalFullBackupSeed{
				DatabaseID: physical.Id,
				StorageID:  team.StorageID,
				CreatedAt:  physicalTime,
				StartLSN:   new("0/1000000"), StopLSN: new("0/2000000"),
				BackupSizeMb: new(6.1),
				RawSizeMb:    new(9.1),
			},
		)
		selectedSize := float64(6)
		selectedRaw := float64(9)
		if firstFull.ID.String() < secondFull.ID.String() {
			selectedSize = 7
			selectedRaw = 10
		}
		team.App.InsertPhysicalFullBackup(
			t,
			apptest.PhysicalFullBackupSeed{
				DatabaseID:   physical.Id,
				StorageID:    team.StorageID,
				CreatedAt:    physicalTime.Add(time.Minute),
				Status:       "ERROR",
				BackupSizeMb: new(900.0),
			},
		)
		team.App.InsertPhysicalIncrementalBackup(
			t,
			apptest.PhysicalIncrementalBackupSeed{
				DatabaseID:       physical.Id,
				StorageID:        team.StorageID,
				RootFullBackupID: firstFull.ID,
				CreatedAt:        physicalTime.Add(2 * time.Minute),
				StartLSN:         new("0/2000000"), StopLSN: new("0/3000000"),
				BackupSizeMb: new(800.0),
			},
		)

		config, err := team.Owner.Client.UpdateVerificationConfigWithResponse(
			t.Context(),
			logical.Id,
			apiclient.VerificationConfigInput{
				IsAutomaticVerificationEnabled: true,
				ScheduleType:                   apiclient.VerificationScheduleTypeINTERVAL,
				Schedule: apiclient.Schedule{
					Type:      apiclient.ScheduleTypeDAILY,
					TimeOfDay: new("10:00"),
				},
				NotificationTriggers: []apiclient.VerificationNotificationTrigger{},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, config.JSON200, string(config.Body))

		agent := team.App.CreateVerificationAgent(t, team.Admin)
		heartbeat, err := team.App.NewAgentClient(t, agent).
			RecordHeartbeatWithResponse(t.Context(), agent.Agent.ID, agentclient.HeartbeatRequest{MaxCPU: 2, MaxRAMMb: 512, MaxDiskGb: 7, MaxConcurrentJobs: 3, CurrentVerificationIDs: []uuid.UUID{}})
		require.NoError(t, err)
		require.NotNil(t, heartbeat.JSON200)

		team.App.RunJobPass(t, "telemetry_ping")
		team.App.Clock.Advance(time.Minute)
		team.App.RunJobPass(t, "telemetry_ping")
		reports := receiver.getReports()
		require.Len(t, reports, 1)
		entries, isArray := reports[0]["databases"].([]any)
		require.True(t, isArray)
		require.Len(t, entries, 3)

		hasLogicalSize, hasPhysicalSize, hasMissingSize := false, false, false
		for _, rawEntry := range entries {
			entry, isObject := rawEntry.(map[string]any)
			require.True(t, isObject)
			switch {
			case entry["type"] == "POSTGRES_PHYSICAL":
				hasPhysicalSize = true
				assert.InDelta(t, selectedSize, entry["backupSizeMb"], 0)
				assert.InDelta(t, selectedRaw, entry["rawSizeMb"], 0)
				assert.Equal(t, "FULL", entry["backupType"])
			case entry["backupSizeMb"] != nil:
				hasLogicalSize = true
				assert.InDelta(t, 1, entry["backupSizeMb"], 0)
				assert.InDelta(t, 3, entry["rawSizeMb"], 0)
				verificationConfig, isVerification := entry["verification"].(map[string]any)
				require.True(t, isVerification)
				assert.Equal(t, "DAILY", verificationConfig["intervalType"])
			default:
				hasMissingSize = true
				assert.NotContains(t, entry, "rawSizeMb")
			}
			assert.Equal(t, "18", entry["version"])
			assert.Equal(t, false, entry["isSshTunnelEnabled"])
		}
		assert.True(t, hasLogicalSize && hasPhysicalSize && hasMissingSize)
		assert.Equal(
			t,
			[]any{
				map[string]any{
					"maxCpu":            float64(2),
					"maxRamMb":          float64(512),
					"maxDiskGb":         float64(7),
					"maxConcurrentJobs": float64(3),
				},
			},
			reports[0]["verificationAgents"],
		)

		encoded, err := json.Marshal(reports[0])
		require.NoError(t, err)
		for _, privateValue := range []string{logical.Id.String(), physical.Id.String(), logical.Name, team.Workspace.Id.String(), containers.PostgresPassword, agent.Token, source.Host} {
			assert.NotContains(t, string(encoded), privateValue)
		}

		team.App.ExecuteSQL(t, "UPDATE databases SET health_status = 'UNAVAILABLE' WHERE id = ?", logical.Id)
		team.App.Clock.Advance(25 * time.Hour)
		team.App.RunJobPass(t, "telemetry_ping")

		availabilityReports := receiver.getReports()
		require.Len(t, availabilityReports, 2)
		availableEntries, isAvailableArray := availabilityReports[1]["databases"].([]any)
		require.True(t, isAvailableArray)
		require.Len(t, availableEntries, 2)
		for _, rawEntry := range availableEntries {
			entry, isObject := rawEntry.(map[string]any)
			require.True(t, isObject)
			if entry["type"] == "POSTGRES_LOGICAL" {
				assert.NotContains(t, entry, "backupSizeMb")
				assert.NotContains(t, entry, "verification")
			}
		}

		team.App.RunJobPass(t, "telemetry_ping")
		reportAt := team.App.Clock.Now().Add(25 * time.Hour)
		boundary := reportAt.Add(-7 * 24 * time.Hour)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases SET health_status = NULL WHERE id IN ?",
			[]uuid.UUID{logical.Id, secondLogical.Id, physical.Id},
		)
		team.App.ExecuteSQL(t, "UPDATE backups_logical SET created_at = ? WHERE id = ?", boundary, logicalBackup.Id)
		team.App.ExecuteSQL(t, "UPDATE databases SET last_backup_time = ? WHERE id = ?", boundary, physical.Id)
		team.App.Clock.Advance(25 * time.Hour)
		team.App.RunJobPass(t, "telemetry_ping")
		boundaryReports := receiver.getReports()
		require.Len(t, boundaryReports, 3)
		boundaryEntries, isBoundaryArray := boundaryReports[2]["databases"].([]any)
		require.True(t, isBoundaryArray)
		require.Len(t, boundaryEntries, 1)
		boundaryEntry, isBoundaryObject := boundaryEntries[0].(map[string]any)
		require.True(t, isBoundaryObject)
		assert.Equal(t, "POSTGRES_LOGICAL", boundaryEntry["type"])

		reportAt = team.App.Clock.Now().Add(25 * time.Hour)
		boundary = reportAt.Add(-7 * 24 * time.Hour)
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_logical SET created_at = ? WHERE id = ?",
			boundary.Add(-time.Second),
			logicalBackup.Id,
		)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases SET last_backup_time = ? WHERE id = ?",
			boundary.Add(time.Second),
			physical.Id,
		)
		team.App.Clock.Advance(25 * time.Hour)
		team.App.RunJobPass(t, "telemetry_ping")
		boundaryReports = receiver.getReports()
		require.Len(t, boundaryReports, 4)
		boundaryEntries, isBoundaryArray = boundaryReports[3]["databases"].([]any)
		require.True(t, isBoundaryArray)
		require.Len(t, boundaryEntries, 1)
		boundaryEntry, isBoundaryObject = boundaryEntries[0].(map[string]any)
		require.True(t, isBoundaryObject)
		assert.Equal(t, "POSTGRES_PHYSICAL", boundaryEntry["type"])
	})
}
