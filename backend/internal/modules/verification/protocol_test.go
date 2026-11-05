package verification_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/agentclient"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func heartbeatBody(ids []uuid.UUID) agentclient.HeartbeatRequest {
	return agentclient.HeartbeatRequest{
		MaxCPU:                 4,
		MaxRAMMb:               8192,
		MaxDiskGb:              10,
		MaxConcurrentJobs:      2,
		CurrentVerificationIDs: ids,
	}
}

func claimBody(diskGb int) agentclient.ClaimRequest {
	return agentclient.ClaimRequest{
		Capacity: agentclient.AgentCapacity{MaxCPU: 4, MaxRAMMb: 8192, MaxDiskGb: diskGb, MaxConcurrentJobs: 2},
	}
}

func (team verificationTeam) listRunningVerificationIDsOwnedBy(t *testing.T, agentID uuid.UUID) []uuid.UUID {
	t.Helper()
	ownedVerificationIDs := []uuid.UUID{}

	for _, verification := range team.list(t) {
		isRunning := verification.Status == apiclient.VerificationStatusRUNNING
		if isRunning && verification.AgentId != nil && *verification.AgentId == agentID {
			ownedVerificationIDs = append(ownedVerificationIDs, verification.Id)
		}
	}

	return ownedVerificationIDs
}

func claim(t *testing.T, team verificationTeam, agent apptest.CreatedVerificationAgent) uuid.UUID {
	t.Helper()
	protocol := newProtocolAgent(t, team.App, agent)
	protocol.requireAcceptedHeartbeat(team.listRunningVerificationIDsOwnedBy(t, agent.Agent.ID))

	status, body := protocol.claimVerification(claimBody(25))
	require.Equal(t, http.StatusOK, status, string(body))
	var assignment struct {
		VerificationID uuid.UUID `json:"verificationId"`
	}
	require.NoError(t, json.Unmarshal(body, &assignment))
	return assignment.VerificationID
}

func TestVerificationProtocol(t *testing.T) {
	t.Parallel()

	t.Run("updates capacity and the last seen time when an agent heartbeats twice", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		agent := app.CreateVerificationAgent(t, admin)

		for range 2 {
			status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
			require.Equal(t, http.StatusOK, status, string(body))
			app.Clock.Advance(time.Second)
		}

		listed, err := admin.Client.ListVerificationAgentsWithResponse(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, listed.JSON200)
		require.Len(t, listed.JSON200.Items, 1)
		persisted := listed.JSON200.Items[0]

		assert.Equal(t, 4, persisted.MaxCpu)
		assert.Equal(t, 8192, persisted.MaxRamMb)
		assert.Equal(t, 10, persisted.MaxDiskGb)
		assert.Equal(t, 2, persisted.MaxConcurrentJobs)
		require.NotNil(t, persisted.LastSeenAt)
	})
	t.Run("invalidates the previous token when an admin rotates it", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		agent := app.CreateVerificationAgent(t, admin)

		rotated, err := admin.Client.RotateVerificationAgentTokenWithResponse(t.Context(), agent.Agent.ID)
		require.NoError(t, err)
		require.NotNil(t, rotated.JSON200)

		status, _ := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
		assert.Equal(t, http.StatusUnauthorized, status)

		agent.Token = rotated.JSON200.Token
		status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
		assert.Equal(t, http.StatusOK, status, string(body))
	})
	t.Run("refuses credentials and capacity that are invalid without changing the agent", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		agent := app.CreateVerificationAgent(t, admin)

		var refusedDetail string
		for index, token := range []string{"", "invalid", admin.Token} {
			invalid := agent
			invalid.Token = token
			status, body := newProtocolAgent(t, app, invalid).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
			problem := requireAgentProblem(
				t,
				agentResponse{Status: status, Body: body},
				agentclient.Problem{Status: http.StatusUnauthorized, Code: agentclient.ProblemCodeUnauthenticated},
			)
			if index == 0 {
				refusedDetail = problem.Detail
			}
			assert.Equal(t, refusedDetail, problem.Detail)
		}

		status, body := newProtocolAgent(t, app, agent).sendInvalidHeartbeat("invalid")
		malformedIDProblem := requireAgentProblem(
			t,
			agentResponse{Status: status, Body: body},
			agentclient.Problem{Status: http.StatusUnauthorized, Code: agentclient.ProblemCodeUnauthenticated},
		)
		assert.Equal(t, refusedDetail, malformedIDProblem.Detail)

		negative := heartbeatBody(nil)
		negative.MaxRAMMb = -1
		status, _ = newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, negative)
		assert.Equal(t, http.StatusBadRequest, status)

		agentClient := app.NewSessionClient(t, agent.Token)
		listed, err := agentClient.ListVerificationAgentsWithResponse(t.Context(), nil)
		require.NoError(t, err)
		requireProblem(
			t,
			listed.StatusCode(),
			listed.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		otherAgent := app.CreateVerificationAgent(t, admin)
		status, _ = newProtocolAgent(t, app, agent).recordHeartbeat(otherAgent.Agent.ID, heartbeatBody(nil))
		assert.Equal(t, http.StatusUnauthorized, status)
	})
	t.Run("limits requests per agent without charging invalid UUIDs", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		agent := app.CreateVerificationAgent(t, admin)

		for range 12 {
			status, body := newProtocolAgent(t, app, agent).sendInvalidHeartbeat("not-an-id")
			requireAgentProblem(
				t,
				agentResponse{Status: status, Body: body},
				agentclient.Problem{Status: http.StatusUnauthorized, Code: agentclient.ProblemCodeUnauthenticated},
			)
		}
		for range 10 {
			status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
			require.Equal(t, http.StatusOK, status, string(body))
		}

		status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
		requireAgentProblem(
			t,
			agentResponse{Status: status, Body: body},
			agentclient.Problem{Status: http.StatusTooManyRequests, Code: agentclient.ProblemCodeRateLimited},
		)
	})
	t.Run("returns only engine facts and the disk budget when a pending job fits", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		team.App.ExecuteSQL(t, "UPDATE backups_logical SET backup_raw_db_size_mb=0.01 WHERE id=?", backup.Id)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		protocol := newProtocolAgent(t, team.App, agent)
		protocol.requireAcceptedHeartbeat(nil)

		status, body := protocol.claimVerification(claimBody(5))
		require.Equal(t, http.StatusOK, status, string(body))
		var assignment map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &assignment))

		assert.JSONEq(
			t,
			`{"type":"POSTGRES_LOGICAL","postgresqlLogical":{"version":"18"}}`,
			string(assignment["database"]),
		)
		assert.NotContains(t, string(body), "password")
		assert.NotContains(t, string(body), "host")
		assert.Equal(t, apiclient.VerificationStatusRUNNING, team.read(t, verification.Id).Status)
	})
	t.Run("returns no job when the pending backup exceeds the agent budget", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		protocol := newProtocolAgent(t, team.App, agent)
		protocol.requireAcceptedHeartbeat(nil)

		status, body := protocol.claimVerification(claimBody(1))
		assert.Equal(t, http.StatusNoContent, status, string(body))
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, verification.Id).Status)
	})
	t.Run("stores results and table counts when the owner reports completion", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		assert.Equal(t, verification.Id, claim(t, team, agent))

		report := agentclient.VerificationReport{
			Status:                  "COMPLETED",
			PgRestoreExitCode:       new(1),
			RestoreDurationMs:       new(int64(123)),
			VerifyDurationMs:        new(int64(45)),
			DBSizeBytesAfterRestore: new(int64(1024 * 1024 * 1024)),
			TableCount:              new(2),
			SchemaCount:             new(1),
			TableStats: []agentclient.ReportTableStat{
				{SchemaName: "public", Name: "first", RowCount: 7},
				{SchemaName: "public", Name: "second", RowCount: 9},
			},
		}

		status, body := newProtocolAgent(t, team.App, agent).reportVerification(verification.Id, report)
		require.Equal(t, http.StatusNoContent, status, string(body))

		completed := team.read(t, verification.Id)
		assert.Equal(t, apiclient.VerificationStatusCOMPLETED, completed.Status)
		require.Len(t, completed.TableStats, 2)
		assert.Equal(t, int64(7), completed.TableStats[0].RowCount)
		assert.Equal(t, 1, *completed.PgRestoreExitCode)
		assert.Equal(t, int64(123), *completed.RestoreDurationMs)
		assert.Equal(t, int64(45), *completed.VerifyDurationMs)
		assert.Equal(t, int64(1024*1024*1024), *completed.RestoredDatabaseSizeBytes)

		listed, err := team.Owner.Client.ListVerificationsWithResponse(
			t.Context(),
			&apiclient.ListVerificationsParams{DatabaseId: team.Database.Id, Limit: nil, After: nil},
		)
		require.NoError(t, err)
		assert.NotContains(t, string(listed.Body), "tableStats")
	})
	t.Run("returns gone when a completed report no longer owns a running verification", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)

		canceled, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		require.NotNil(t, canceled.JSON200)

		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).reportVerification(verification.Id, agentclient.VerificationReport{Status: "COMPLETED"})
		assert.Equal(t, http.StatusGone, status, string(body))
		requireAgentProblem(
			t,
			agentResponse{Status: status, Body: body},
			agentclient.Problem{Status: http.StatusGone, Code: agentclient.ProblemCodeVerificationGone},
		)

		status, _ = newProtocolAgent(
			t,
			team.App,
			agent,
		).reportVerification(uuid.New(), agentclient.VerificationReport{Status: "COMPLETED"})
		assert.Equal(t, http.StatusGone, status)
	})
	t.Run("tells the agent to abort when a running verification is cancelled", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)

		canceled, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		require.NotNil(t, canceled.JSON200)
		assert.Equal(t, apiclient.VerificationStatusCANCELED, canceled.JSON200.Status)

		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).recordHeartbeat(agent.Agent.ID, heartbeatBody([]uuid.UUID{verification.Id}))
		require.Equal(t, http.StatusOK, status, string(body))

		var heartbeat struct {
			AbortIDs []uuid.UUID `json:"abortVerificationIds"`
		}
		require.NoError(t, json.Unmarshal(body, &heartbeat))
		assert.Equal(t, []uuid.UUID{verification.Id}, heartbeat.AbortIDs)
	})
}
