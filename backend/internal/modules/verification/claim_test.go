package verification_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationClaimBudget(t *testing.T) {
	t.Parallel()

	t.Run("skips an oversized oldest candidate and deducts running jobs when choosing work", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		oversized := team.start(t, backup.Id)
		team.App.ExecuteSQL(t, "UPDATE verification_runs SET trigger='SCHEDULED' WHERE id=?", oversized.Id)
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_logical SET backup_size_mb=9000,backup_raw_db_size_mb=1 WHERE id=?",
			backup.Id,
		)

		smaller := team.backup(t)
		team.App.WaitForBackupRuns(t)
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_logical SET backup_size_mb=100,backup_raw_db_size_mb=100 WHERE id=?",
			smaller.Id,
		)
		candidate := team.start(t, smaller.Id)

		agent := team.App.CreateVerificationAgent(t, team.Admin)
		protocol := newProtocolAgent(t, team.App, agent)
		protocol.requireAcceptedHeartbeat(nil)
		status, body := protocol.claimVerification(claimBody(5))
		require.Equal(t, http.StatusOK, status, string(body))
		assert.Equal(t, apiclient.VerificationStatusRUNNING, team.read(t, candidate.Id).Status)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, oversized.Id).Status)

		team.App.ExecuteSQL(
			t,
			"UPDATE backups_logical SET backup_size_mb=4000,backup_raw_db_size_mb=1 WHERE id=?",
			backup.Id,
		)
		status, body = protocol.claimVerification(claimBody(6))
		assert.Equal(t, http.StatusNoContent, status, string(body))
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, oversized.Id).Status)
	})
	t.Run("leaves a claimed job with its owner when another agent claims", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		owner := team.App.CreateVerificationAgent(t, team.Admin)
		other := team.App.CreateVerificationAgent(t, team.Admin)

		assert.Equal(t, verification.Id, claim(t, team, owner))
		otherProtocol := newProtocolAgent(t, team.App, other)
		otherProtocol.requireAcceptedHeartbeat(nil)
		status, body := otherProtocol.claimVerification(claimBody(25))
		assert.Equal(t, http.StatusNoContent, status, string(body))
		assert.Equal(t, owner.Agent.ID, *team.read(t, verification.Id).AgentId)
	})
	t.Run("forwards the backup TimescaleDB version when assigning work", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		team.App.ExecuteSQL(t, "UPDATE backups_logical SET timescaledb_version='2.25.0' WHERE id=?", backup.Id)
		team.start(t, backup.Id)

		agent := team.App.CreateVerificationAgent(t, team.Admin)
		protocol := newProtocolAgent(t, team.App, agent)
		protocol.requireAcceptedHeartbeat(nil)
		status, body := protocol.claimVerification(claimBody(25))
		require.Equal(t, http.StatusOK, status, string(body))
		assert.Contains(t, string(body), `"timescaledbVersion":"2.25.0"`)
	})
}
