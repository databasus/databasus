package e2e_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The agent of each scenario creates a Docker network, and Docker's address pools run out when all scenarios start at once.
func TestVerifyBackup(t *testing.T) {
	t.Parallel()

	for _, version := range []struct {
		major string
		image string
	}{
		{"12", "postgres:12"}, {"13", "postgres:13"}, {"14", "postgres:14"}, {"15", "postgres:15"}, {"16", "postgres:16"}, {"17", "postgres:17"}, {"18", "postgres:18"},
	} {
		t.Run("completes with an encrypted backup on postgres "+version.major, func(t *testing.T) {
			endpoint := containers.StartPostgres(t, version.image)
			verifyBackup(t, backupScenario{endpoint: endpoint})
		})
	}
	t.Run("completes with a timescaledb hypertable and its chunks", func(t *testing.T) {
		endpoint := containers.StartTimescaleDB(t, timescaleImage)
		verifyBackup(t, backupScenario{endpoint: endpoint, isTimescale: true})
	})
	t.Run("completes when the backup includes only the public schema", func(t *testing.T) {
		endpoint := containers.StartPostgres(t, "postgres:18")
		verifyBackup(t, backupScenario{endpoint: endpoint, isPublicOnly: true})
	})
	t.Run("fails after one attempt when the backup is broken", func(t *testing.T) {
		endpoint := containers.StartPostgres(t, "postgres:18")
		verifyBackup(t, backupScenario{endpoint: endpoint, isBroken: true})
	})
	t.Run("fails after one attempt when the disk budget is exceeded", func(t *testing.T) {
		endpoint := containers.StartPostgres(t, "postgres:18")
		verifyBackup(t, backupScenario{endpoint: endpoint, diskCapMb: 1})
	})
	t.Run("completes after removing a network left by an earlier attempt", func(t *testing.T) {
		endpoint := containers.StartPostgres(t, "postgres:18")
		verifyBackup(t, backupScenario{endpoint: endpoint, isRetry: true})
	})
}

type backupScenario struct {
	endpoint     containers.Endpoint
	isTimescale  bool
	isPublicOnly bool
	isBroken     bool
	isRetry      bool
	diskCapMb    int
}

func verifyBackup(t *testing.T, scenario backupScenario) {
	t.Helper()
	team := enginetest.NewTeam(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
	versionResponse, err := team.Owner.Client.GetVersionWithResponse(t.Context())
	require.NoError(t, err)
	require.Equal(t, serverVersion, versionResponse.JSON200.Version)

	settings := seedSource(t, scenario.endpoint, scenario.isTimescale)
	if scenario.isPublicOnly {
		settings.IncludeSchemas = []string{"public"}
	}
	database := team.CreateDatabase(t, settings)
	encryption := apiclient.BackupEncryptionENCRYPTED
	if scenario.isBroken {
		encryption = apiclient.BackupEncryptionNONE
	}

	backup := team.BackUp(t, database.Id, encryption)
	if scenario.isBroken {
		team.App.DamageLocalBackup(t, backup.Id, []byte("not a pg_dump archive\n"))
	}

	credentials := team.App.CreateVerificationAgent(t, team.Admin)
	engine := newEngine(t, credentials.Agent.ID)
	gate := newGateway(t, team.App.URL, gatewayOptions{isClaimHeld: scenario.isRetry, diskCapMb: scenario.diskCapMb})
	agent := newAgentProcess(t, currentBinaryDirectory)

	var verificationID uuid.UUID
	if scenario.isRetry {
		agent.start(t, gate.url, credentials)
		waitForHeartbeat(t, team.App, team.Admin, credentials.Agent.ID, time.Time{})
		verificationID = team.App.EnqueueVerification(t, team.Owner, backup.Id)
		createNetwork(t, engine, credentials.Agent.ID, verificationID)
		gate.releaseClaims()
	} else {
		verificationID = team.App.EnqueueVerification(t, team.Owner, backup.Id)
		agent.start(t, gate.url, credentials)
	}

	verification := waitForVerification(t, team.App, team.Owner, verificationID)
	if scenario.isBroken || scenario.diskCapMb > 0 {
		require.Equal(t, "FAILED", verification.Status, fmt.Sprint(verification.FailMessage))
		assert.Equal(t, 1, verification.AttemptCount)
		require.NotNil(t, verification.FailMessage)
		assert.NotEmpty(t, *verification.FailMessage)
		if scenario.diskCapMb > 0 {
			assert.Contains(t, strings.ToLower(*verification.FailMessage), "disk budget")
		}
	} else {
		requireCompleted(t, verification)
		if scenario.isTimescale {
			var hasSensorTable bool
			var chunkRows int64
			for _, table := range verification.TableStats {
				if table.Name == "sensor_data" {
					hasSensorTable = true
				}
				if strings.HasPrefix(table.SchemaName, "_timescaledb") && strings.HasPrefix(table.Name, "_hyper_") {
					chunkRows += table.RowCount
				}
			}
			assert.True(t, hasSensorTable)
			assert.Positive(t, chunkRows)
		} else {
			require.NotNil(t, verification.TableCount)
			assert.Equal(t, 2, *verification.TableCount)
			require.NotNil(t, verification.SchemaCount)
			assert.Equal(t, 1, *verification.SchemaCount)
			tableRows := map[string]int64{}
			for _, table := range verification.TableStats {
				tableRows[table.Name] = table.RowCount
			}
			assert.Equal(t, int64(3), tableRows["t_a"])
			assert.Equal(t, int64(5), tableRows["t_b"])
		}
	}

	requireNoLeftovers(t, engine, credentials.Agent.ID)

	if scenario.isBroken {
		details, err := team.Owner.Client.GetVerificationWithResponse(t.Context(), verificationID)
		require.NoError(t, err)
		require.NotNil(t, details.JSON200)
		require.NotNil(t, details.JSON200.FailureDiagnostic)
		assert.Contains(t, details.JSON200.FailureDiagnostic.Output, "pg_restore:")
		assert.False(t, details.JSON200.FailureDiagnostic.IsTruncated)
	}
}

func requireCompleted(t *testing.T, verification apptest.Verification) {
	t.Helper()
	require.Equal(t, "COMPLETED", verification.Status, fmt.Sprint(verification.FailMessage))
	require.NotNil(t, verification.PgRestoreExitCode)
	assert.Zero(t, *verification.PgRestoreExitCode)
	require.NotNil(t, verification.DBSizeBytesAfterRestore)
	assert.Positive(t, *verification.DBSizeBytesAfterRestore)
}
