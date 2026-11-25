package verification_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentclient "databasus-backend/internal/lib/agentclient"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// getIndexedBackupSizes returns the sizes of the backup-verification spec's examples: a 10000 MB
// file, a 20000 MB database and its five largest indexes.
func getIndexedBackupSizes() backupSizes {
	return backupSizes{archiveMb: 10000, databaseMb: 20000, indexSizesMb: "{3000,2000,1000,500,100}"}
}

type backupSizes struct {
	archiveMb  float64
	databaseMb float64
	// indexSizesMb is the SQL array literal of the recorded index sizes, empty for none recorded.
	indexSizesMb string
}

type claimedCapacity struct {
	maxCPU            int
	maxConcurrentJobs int
	maxDiskGb         int
}

func TestVerificationClaimRestorePlan(t *testing.T) {
	t.Parallel()

	t.Run("assigns the most parallel jobs and the disk budget that fit the agent", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		cases := []struct {
			name                 string
			capacity             claimedCapacity
			sizes                backupSizes
			timescaledbVersion   string
			expectedParallelJobs int
			expectedDiskBudgetMb float64
		}{
			{
				name:                 "every job of the per-job CPU budget when the disk holds them",
				capacity:             claimedCapacity{maxCPU: 4, maxConcurrentJobs: 1, maxDiskGb: 100},
				sizes:                getIndexedBackupSizes(),
				expectedParallelJobs: 4,
				expectedDiskBudgetMb: 38036,
			},
			{
				name:                 "fewer jobs when the disk holds fewer index builds",
				capacity:             claimedCapacity{maxCPU: 4, maxConcurrentJobs: 1, maxDiskGb: 36},
				sizes:                getIndexedBackupSizes(),
				expectedParallelJobs: 2,
				expectedDiskBudgetMb: 36536,
			},
			{
				name:                 "the per-job CPU budget of an agent with two concurrent jobs",
				capacity:             claimedCapacity{maxCPU: 8, maxConcurrentJobs: 2, maxDiskGb: 100},
				sizes:                getIndexedBackupSizes(),
				expectedParallelJobs: 4,
				expectedDiskBudgetMb: 38036,
			},
			{
				name:                 "one job when there are fewer CPUs than concurrent jobs",
				capacity:             claimedCapacity{maxCPU: 1, maxConcurrentJobs: 2, maxDiskGb: 100},
				sizes:                getIndexedBackupSizes(),
				expectedParallelJobs: 1,
				expectedDiskBudgetMb: 34536,
			},
			{
				name:                 "at most eight jobs, counting all five recorded indexes",
				capacity:             claimedCapacity{maxCPU: 16, maxConcurrentJobs: 1, maxDiskGb: 100},
				sizes:                getIndexedBackupSizes(),
				expectedParallelJobs: 8,
				expectedDiskBudgetMb: 38136,
			},
			{
				name:                 "one job for a timescaledb backup",
				capacity:             claimedCapacity{maxCPU: 4, maxConcurrentJobs: 1, maxDiskGb: 100},
				sizes:                getIndexedBackupSizes(),
				timescaledbVersion:   "2.17.0",
				expectedParallelJobs: 1,
				expectedDiskBudgetMb: 34536,
			},
			{
				name:                 "one job and the clamped gap for a backup without recorded index sizes",
				capacity:             claimedCapacity{maxCPU: 4, maxConcurrentJobs: 1, maxDiskGb: 100},
				sizes:                backupSizes{archiveMb: 10000, databaseMb: 20000, indexSizesMb: ""},
				expectedParallelJobs: 1,
				expectedDiskBudgetMb: 35120,
			},
			{
				name:                 "one job and the gap cap for a backup of an unknown database size",
				capacity:             claimedCapacity{maxCPU: 4, maxConcurrentJobs: 1, maxDiskGb: 100},
				sizes:                backupSizes{archiveMb: 10000, databaseMb: 0, indexSizesMb: "{3000,2000}"},
				expectedParallelJobs: 1,
				expectedDiskBudgetMb: 15120,
			},
			{
				name:                 "every job for a backup with no indexes",
				capacity:             claimedCapacity{maxCPU: 4, maxConcurrentJobs: 1, maxDiskGb: 100},
				sizes:                backupSizes{archiveMb: 10000, databaseMb: 20000, indexSizesMb: "{}"},
				expectedParallelJobs: 4,
				expectedDiskBudgetMb: 31536,
			},
		}

		// The cases share one team and run in order: each cancels its verification before the next
		// claims, because a claim takes the oldest pending verification.
		for _, planCase := range cases {
			t.Run(planCase.name, func(t *testing.T) {
				verification := team.startSizedVerification(t, planCase.sizes)
				if planCase.timescaledbVersion != "" {
					team.App.ExecuteSQL(t, "UPDATE backups_logical SET timescaledb_version = ? WHERE id = ?",
						planCase.timescaledbVersion, verification.BackupId)
				}

				assignment := requireAssignment(t, newClaimingAgent(t, team), planCase.capacity)

				assert.Equal(t, verification.Id, assignment.VerificationID)
				assert.Equal(t, planCase.expectedParallelJobs, assignment.ParallelJobs)
				assert.InDelta(t, planCase.expectedDiskBudgetMb, assignment.MaxContainerDiskMb, 0)
				team.cancel(t, verification.Id)
			})
		}
	})

	t.Run("leaves a verification pending when not even one job fits the disk", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.startSizedVerification(t, getIndexedBackupSizes())
		attemptCount := team.read(t, verification.Id).AttemptCount

		status, body := newClaimingAgent(t, team).claimVerification(
			newClaimRequest(claimedCapacity{maxCPU: 4, maxConcurrentJobs: 1, maxDiskGb: 33}),
		)

		assert.Equal(t, http.StatusNoContent, status, string(body))
		pendingVerification := team.read(t, verification.Id)
		assert.Equal(t, apiclient.VerificationStatusPENDING, pendingVerification.Status)
		assert.Nil(t, pendingVerification.AgentId)
		assert.Equal(t, attemptCount, pendingVerification.AttemptCount)
	})

	t.Run("assigns the jobs the disk left by a running verification holds", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		agent := newClaimingAgent(t, team)
		team.runVerificationHoldingDisk(t, agent, 65000)
		waiting := team.startSizedVerification(t, getIndexedBackupSizes())

		assignment := requireAssignment(t, agent, claimedCapacity{maxCPU: 8, maxConcurrentJobs: 2, maxDiskGb: 100})

		assert.Equal(t, waiting.Id, assignment.VerificationID)
		assert.Equal(t, 2, assignment.ParallelJobs)
		assert.InDelta(t, 36536, assignment.MaxContainerDiskMb, 0)
	})

	t.Run("assigns a waiting verification after the running one completes", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		agent := newClaimingAgent(t, team)
		runningID := team.runVerificationHoldingDisk(t, agent, 70000)
		waiting := team.startSizedVerification(t, getIndexedBackupSizes())
		capacity := claimedCapacity{maxCPU: 8, maxConcurrentJobs: 2, maxDiskGb: 100}

		status, body := agent.claimVerification(newClaimRequest(capacity))
		require.Equal(t, http.StatusNoContent, status, string(body))
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, waiting.Id).Status)

		status, body = agent.reportVerification(
			runningID,
			agentclient.VerificationReport{Status: agentclient.VerificationReportStatusCOMPLETED},
		)
		require.Equal(t, http.StatusNoContent, status, string(body))

		assignment := requireAssignment(t, agent, capacity)
		assert.Equal(t, waiting.Id, assignment.VerificationID)
		assert.Equal(t, 4, assignment.ParallelJobs)
	})

	t.Run("assigns a waiting verification after the running one is cancelled", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		agent := newClaimingAgent(t, team)
		runningID := team.runVerificationHoldingDisk(t, agent, 70000)
		waiting := team.startSizedVerification(t, getIndexedBackupSizes())
		capacity := claimedCapacity{maxCPU: 8, maxConcurrentJobs: 2, maxDiskGb: 100}

		status, body := agent.claimVerification(newClaimRequest(capacity))
		require.Equal(t, http.StatusNoContent, status, string(body))

		team.cancel(t, runningID)

		assignment := requireAssignment(t, agent, capacity)
		assert.Equal(t, waiting.Id, assignment.VerificationID)
		assert.Equal(t, 4, assignment.ParallelJobs)
		assert.InDelta(t, 38036, assignment.MaxContainerDiskMb, 0)
	})
}

// startSizedVerification backs up the team's database, gives the backup sizes the API cannot set and
// starts its verification.
func (team verificationTeam) startSizedVerification(t *testing.T, sizes backupSizes) apiclient.Verification {
	t.Helper()

	return team.startSizedVerificationOf(t, team.Database.Id, sizes)
}

func (team verificationTeam) startSizedVerificationOf(
	t *testing.T,
	databaseID uuid.UUID,
	sizes backupSizes,
) apiclient.Verification {
	t.Helper()

	backup := team.BackUp(t, databaseID, apiclient.BackupEncryptionNONE)
	team.App.WaitForBackupRuns(t)

	if sizes.indexSizesMb == "" {
		team.App.ExecuteSQL(t, `UPDATE backups_logical SET backup_size_mb = ?, backup_raw_db_size_mb = ?,
			backup_largest_index_sizes_mb = NULL WHERE id = ?`, sizes.archiveMb, sizes.databaseMb, backup.Id)
	} else {
		team.App.ExecuteSQL(t, `UPDATE backups_logical SET backup_size_mb = ?, backup_raw_db_size_mb = ?,
			backup_largest_index_sizes_mb = ?::double precision[] WHERE id = ?`,
			sizes.archiveMb, sizes.databaseMb, sizes.indexSizesMb, backup.Id)
	}

	return team.start(t, backup.Id)
}

// runVerificationHoldingDisk has the agent claim a small verification of a second database, because
// a database runs one verification at a time, and sets the disk budget it holds, which a real claim
// gets only from a backup of that size.
func (team verificationTeam) runVerificationHoldingDisk(
	t *testing.T,
	agent protocolAgent,
	reservedDiskMb int64,
) uuid.UUID {
	t.Helper()

	running := team.startSizedVerificationOf(
		t,
		team.createSecondDatabase(t),
		backupSizes{archiveMb: 1, databaseMb: 1, indexSizesMb: "{}"},
	)
	assignment := requireAssignment(t, agent, claimedCapacity{maxCPU: 8, maxConcurrentJobs: 2, maxDiskGb: 100})
	require.Equal(t, running.Id, assignment.VerificationID)
	team.App.ExecuteSQL(t, "UPDATE verification_runs SET reserved_disk_mb = ? WHERE id = ?", reservedDiskMb, running.Id)

	return running.Id
}

func (team verificationTeam) createSecondDatabase(t *testing.T) uuid.UUID {
	t.Helper()

	settings, err := team.Database.Settings.AsPostgresqlLogicalSettings()
	require.NoError(t, err)
	settings.Password = new(containers.PostgresPassword)

	return team.CreateDatabase(t, settings).Id
}

func (team verificationTeam) cancel(t *testing.T, verificationID uuid.UUID) {
	t.Helper()

	response, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verificationID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func newClaimingAgent(t *testing.T, team verificationTeam) protocolAgent {
	t.Helper()

	agent := newProtocolAgent(t, team.App, team.App.CreateVerificationAgent(t, team.Admin))
	agent.requireAcceptedHeartbeat(nil)

	return agent
}

func newClaimRequest(capacity claimedCapacity) agentclient.ClaimRequest {
	return agentclient.ClaimRequest{Capacity: agentclient.AgentCapacity{
		MaxCPU:            capacity.maxCPU,
		MaxRAMMb:          8192,
		MaxDiskGb:         capacity.maxDiskGb,
		MaxConcurrentJobs: capacity.maxConcurrentJobs,
	}}
}

func requireAssignment(
	t *testing.T,
	agent protocolAgent,
	capacity claimedCapacity,
) agentclient.VerificationAssignment {
	t.Helper()

	status, body := agent.claimVerification(newClaimRequest(capacity))
	require.Equal(t, http.StatusOK, status, string(body))

	var assignment agentclient.VerificationAssignment
	require.NoError(t, json.Unmarshal(body, &assignment))

	return assignment
}
