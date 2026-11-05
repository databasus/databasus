package verification_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/agentclient"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationReportRetries(t *testing.T) {
	t.Parallel()

	t.Run("requeues twice and fails at the third attempt when an agent cannot prepare the restore", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)

		for attempt := 1; attempt <= 3; attempt++ {
			claim(t, team, agent)
			status, body := newProtocolAgent(
				t,
				team.App,
				agent,
			).reportVerification(verification.Id, agentclient.VerificationReport{Status: "FAILED", FailMessage: new("container could not start")})
			require.Equal(t, http.StatusNoContent, status, string(body))
			persisted := team.read(t, verification.Id)
			if attempt < 3 {
				assert.Equal(t, apiclient.VerificationStatusPENDING, persisted.Status)
				assert.Equal(t, attempt+1, persisted.AttemptCount)
				assert.Nil(t, persisted.AgentId)
				assert.Nil(t, persisted.StartedAt)
			} else {
				assert.Equal(t, apiclient.VerificationStatusFAILED, persisted.Status)
				assert.Equal(t, "container could not start", *persisted.FailMessage)
			}
		}
	})
	for _, failure := range []struct {
		name   string
		report agentclient.VerificationReport
	}{
		{name: "a rejected backup", report: agentclient.VerificationReport{Status: "FAILED", PgRestoreExitCode: new(1), FailMessage: new("archive rejected")}},
		{name: "a disk limit", report: agentclient.VerificationReport{Status: "FAILED", FailureKind: new("DISK_LIMIT_EXCEEDED"), FailMessage: new("disk budget exceeded")}},
		{name: "a restored database below the size guard", report: agentclient.VerificationReport{Status: "COMPLETED", DBSizeBytesAfterRestore: new(int64(1))}},
	} {
		t.Run("fails without retry when the agent reports "+failure.name, func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			team.App.ExecuteSQL(t, "UPDATE backups_logical SET backup_raw_db_size_mb=100 WHERE id=?", backup.Id)
			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			claim(t, team, agent)

			status, body := newProtocolAgent(t, team.App, agent).reportVerification(verification.Id, failure.report)
			require.Equal(t, http.StatusNoContent, status, string(body))

			persisted := team.read(t, verification.Id)
			assert.Equal(t, apiclient.VerificationStatusFAILED, persisted.Status)
			assert.Equal(t, 1, persisted.AttemptCount)
			assert.NotNil(t, persisted.FinishedAt)
			if failure.report.FailMessage != nil {
				assert.Equal(t, *failure.report.FailMessage, *persisted.FailMessage)
			}
		})
	}
}

func TestVerificationHeartbeatReconciliation(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name       string
		age        time.Duration
		isReported bool
		attempts   int
		status     apiclient.VerificationStatus
	}{
		{name: "requeues when a live agent drops an old job", age: 91 * time.Second, isReported: false, attempts: 1, status: apiclient.VerificationStatusPENDING},
		{name: "keeps running when the agent still reports the job", age: 91 * time.Second, isReported: true, attempts: 1, status: apiclient.VerificationStatusRUNNING},
		{name: "keeps running when the claim is within its grace period", age: time.Minute, isReported: false, attempts: 1, status: apiclient.VerificationStatusRUNNING},
		{name: "fails when a dropped job has exhausted its attempts", age: 91 * time.Second, isReported: false, attempts: 3, status: apiclient.VerificationStatusFAILED},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			claim(t, team, agent)

			team.App.ExecuteSQL(
				t,
				"UPDATE verification_runs SET attempt_count=? WHERE id=?",
				scenario.attempts,
				verification.Id,
			)
			team.App.Clock.Advance(scenario.age)

			var ids []uuid.UUID
			if scenario.isReported {
				ids = []uuid.UUID{verification.Id}
			}

			status, body := newProtocolAgent(t, team.App, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(ids))
			require.Equal(t, http.StatusOK, status, string(body))
			assert.Equal(t, scenario.status, team.read(t, verification.Id).Status)
		})
	}
	t.Run("aborts foreign and missing IDs without resurrecting completed work", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		other := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)

		status, body := newProtocolAgent(
			t,
			team.App,
			other,
		).recordHeartbeat(other.Agent.ID, heartbeatBody([]uuid.UUID{verification.Id, uuid.New()}))
		require.Equal(t, http.StatusOK, status, string(body))
		assert.Contains(t, string(body), verification.Id.String())
		assert.Equal(t, apiclient.VerificationStatusRUNNING, team.read(t, verification.Id).Status)

		team.App.ExecuteSQL(
			t,
			"UPDATE verification_runs SET status='COMPLETED',finished_at=? WHERE id=?",
			team.App.Clock.Now(),
			verification.Id,
		)
		team.App.Clock.Advance(2 * time.Minute)

		status, body = newProtocolAgent(
			t,
			team.App,
			agent,
		).recordHeartbeat(agent.Agent.ID, heartbeatBody([]uuid.UUID{verification.Id}))
		require.Equal(t, http.StatusOK, status, string(body))
		assert.Contains(t, string(body), verification.Id.String())
		assert.Equal(t, apiclient.VerificationStatusCOMPLETED, team.read(t, verification.Id).Status)
	})
}

func TestVerificationMixedHeartbeat(t *testing.T) {
	t.Parallel()

	t.Run("reclaims only the dropped verification when an agent continues another one", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		first := team.start(t, backup.Id)
		team.App.ExecuteSQL(t, "UPDATE verification_runs SET trigger='SCHEDULED' WHERE id=?", first.Id)
		second := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		claim(t, team, agent)

		team.App.Clock.Advance(91 * time.Second)
		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).recordHeartbeat(agent.Agent.ID, heartbeatBody([]uuid.UUID{second.Id}))
		require.Equal(t, http.StatusOK, status, string(body))

		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, first.Id).Status)
		assert.Equal(t, apiclient.VerificationStatusRUNNING, team.read(t, second.Id).Status)
	})
}
