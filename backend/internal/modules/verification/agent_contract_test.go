package verification_test

import (
	"fmt"
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

const (
	missingMemoryHeartbeat = `{"maxCpu":1,"maxDiskGb":2,"maxConcurrentJobs":1,"currentVerificationIds":[]}`
	gigabyteOnlyHeartbeat  = `{"maxCpu":1,"maxRamGb":16,"maxDiskGb":2,"maxConcurrentJobs":1,"currentVerificationIds":[]}`
)

func TestVerificationHeartbeatMemory(t *testing.T) {
	t.Parallel()

	for _, memoryMb := range []int{0, 512, 4096} {
		t.Run(fmt.Sprintf("stores %d megabytes when explicitly reported", memoryMb), func(t *testing.T) {
			t.Parallel()

			application := apptest.New(t)
			admin := application.CreateAdmin(t)
			agent := application.CreateVerificationAgent(t, admin)

			heartbeat := heartbeatBody(nil)
			heartbeat.MaxRAMMb = memoryMb
			status, body := newProtocolAgent(t, application, agent).recordHeartbeat(agent.Agent.ID, heartbeat)
			require.Equal(t, http.StatusOK, status, string(body))

			agents := application.ListVerificationAgents(t, admin)
			require.Len(t, agents, 1)
			assert.Equal(t, memoryMb, agents[0].MaxRAMMb)
		})
	}
	t.Run("keeps persisted megabytes when an upgraded instance has no new heartbeat", func(t *testing.T) {
		t.Parallel()

		application := apptest.New(t)
		admin := application.CreateAdmin(t)
		agent := application.CreateVerificationAgent(t, admin)
		application.ExecuteSQL(t, "UPDATE verification_agents SET max_ram_mb = 2048 WHERE id = ?", agent.Agent.ID)
		listed := application.ListVerificationAgents(t, admin)
		require.Len(t, listed, 1)
		assert.Equal(t, 2048, listed[0].MaxRAMMb)
		assert.Nil(t, listed[0].LastSeenAt)
	})

	for _, body := range []string{missingMemoryHeartbeat, gigabyteOnlyHeartbeat} {
		t.Run("preserves capacity and last contact when megabytes are missing", func(t *testing.T) {
			t.Parallel()

			application := apptest.New(t)
			admin := application.CreateAdmin(t)
			agent := application.CreateVerificationAgent(t, admin)
			status, responseBody := newProtocolAgent(
				t,
				application,
				agent,
			).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
			require.Equal(t, http.StatusOK, status, string(responseBody))

			beforeRefusal := application.ListVerificationAgents(t, admin)[0]
			application.Clock.Advance(time.Minute)

			status, responseBody = newProtocolAgent(t, application, agent).sendRawHeartbeat(body)
			requireAgentProblem(
				t,
				agentResponse{Status: status, Body: responseBody},
				agentclient.Problem{Status: http.StatusBadRequest, Code: agentclient.ProblemCodeInvalidRequest},
			)

			afterRefusal := application.ListVerificationAgents(t, admin)[0]
			assert.Equal(t, beforeRefusal, afterRefusal)
		})
	}
}

func TestVerificationAgentAuthentication(t *testing.T) {
	t.Parallel()

	t.Run(
		"returns one detail without updating capacity when credentials do not belong to the named agent",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			agent := app.CreateVerificationAgent(t, admin)
			otherAgent := app.CreateVerificationAgent(t, admin)
			var firstDetail string
			for index, token := range []string{"", "invalid", admin.Token, otherAgent.Token} {
				credentials := agent
				credentials.Token = token
				status, body := newProtocolAgent(
					t,
					app,
					credentials,
				).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
				problem := requireAgentProblem(
					t,
					agentResponse{Status: status, Body: body},
					agentclient.Problem{Status: http.StatusUnauthorized, Code: agentclient.ProblemCodeUnauthenticated},
				)
				if index == 0 {
					firstDetail = problem.Detail
				}
				assert.Equal(t, firstDetail, problem.Detail)
			}

			status, body := newProtocolAgent(t, app, agent).recordHeartbeat(uuid.New(), heartbeatBody(nil))
			unknown := requireAgentProblem(
				t,
				agentResponse{Status: status, Body: body},
				agentclient.Problem{Status: http.StatusUnauthorized, Code: agentclient.ProblemCodeUnauthenticated},
			)
			assert.Equal(t, firstDetail, unknown.Detail)

			listed := app.ListVerificationAgents(t, admin)
			for _, persistedAgent := range listed {
				assert.Nil(t, persistedAgent.LastSeenAt)
			}

			deleted, err := admin.Client.DeleteVerificationAgentWithResponse(t.Context(), agent.Agent.ID)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, deleted.StatusCode())
			status, body = newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
			deletedProblem := requireAgentProblem(
				t,
				agentResponse{Status: status, Body: body},
				agentclient.Problem{Status: http.StatusUnauthorized, Code: agentclient.ProblemCodeUnauthenticated},
			)
			assert.Equal(t, firstDetail, deletedProblem.Detail)
		},
	)
}

func TestVerificationAgentCapacityValidation(t *testing.T) {
	t.Parallel()

	t.Run("preserves capacity when a heartbeat contains a negative value", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		agent := app.CreateVerificationAgent(t, admin)
		status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
		require.Equal(t, http.StatusOK, status, string(body))

		for _, heartbeat := range []agentclient.HeartbeatRequest{
			{MaxRAMMb: -1}, {MaxCPU: -1}, {MaxDiskGb: -1}, {MaxConcurrentJobs: -1},
		} {
			status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeat)
			requireAgentProblem(
				t,
				agentResponse{Status: status, Body: body},
				agentclient.Problem{Status: http.StatusBadRequest, Code: agentclient.ProblemCodeInvalidRequest},
			)
		}

		listed, err := admin.Client.ListVerificationAgentsWithResponse(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, listed.JSON200)
		persistedAgent := listed.JSON200.Items[0]
		assert.Equal(t, 8192, persistedAgent.MaxRamMb)
		assert.Equal(t, 4, persistedAgent.MaxCpu)
		assert.Equal(t, 10, persistedAgent.MaxDiskGb)
		assert.Equal(t, 2, persistedAgent.MaxConcurrentJobs)
	})
	t.Run("leaves work pending when the claim capacity is negative", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		status, body := newProtocolAgent(t, team.App, agent).claimVerification(claimBody(-1))
		requireAgentProblem(
			t,
			agentResponse{Status: status, Body: body},
			agentclient.Problem{Status: http.StatusBadRequest, Code: agentclient.ProblemCodeInvalidRequest},
		)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, verification.Id).Status)
	})
}

func TestVerificationAgentReportValidation(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name   string
		report agentclient.VerificationReport
	}{
		{"keeps running when a report status is unknown", agentclient.VerificationReport{Status: "UNKNOWN"}},
		{"keeps running when a table has no name", agentclient.VerificationReport{Status: agentclient.VerificationReportStatusCOMPLETED, TableStats: []agentclient.ReportTableStat{{SchemaName: "public"}}}},
		{"keeps running when a table has no schema", agentclient.VerificationReport{Status: agentclient.VerificationReportStatusCOMPLETED, TableStats: []agentclient.ReportTableStat{{Name: "customers"}}}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			claim(t, team, agent)

			response, err := team.App.NewAgentClient(t, agent).
				ReportVerificationWithResponse(t.Context(), agent.Agent.ID, verification.Id, scenario.report)
			require.NoError(t, err)
			requireAgentProblem(
				t,
				agentResponse{Status: response.StatusCode(), Body: response.Body},
				agentclient.Problem{Status: http.StatusBadRequest, Code: agentclient.ProblemCodeInvalidRequest},
			)

			assert.Equal(t, apiclient.VerificationStatusRUNNING, team.read(t, verification.Id).Status)
		})
	}
}

func TestVerificationAgentGone(t *testing.T) {
	t.Parallel()

	for _, state := range []string{"cancelled", "reclaimed", "unknown", "deleted backup", "completed"} {
		t.Run("returns gone for stream and report when the verification is "+state, func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			claim(t, team, agent)
			verificationID := verification.Id

			switch state {
			case "cancelled":
				response, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verificationID)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode())
			case "reclaimed":
				team.App.Clock.Advance(91 * time.Second)
				status, body := newProtocolAgent(t, team.App, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
				require.Equal(t, http.StatusOK, status, string(body))
				otherAgent := team.App.CreateVerificationAgent(t, team.Admin)
				require.Equal(t, verificationID, claim(t, team, otherAgent))
				agentOwner := team.read(t, verificationID).AgentId
				require.NotNil(t, agentOwner)
				require.Equal(t, otherAgent.Agent.ID, *agentOwner)
			case "unknown":
				verificationID = uuid.New()
			case "deleted backup":
				response, err := team.Owner.Client.DeleteLogicalBackupWithResponse(t.Context(), backup.Id)
				require.NoError(t, err)
				require.Equal(t, http.StatusNoContent, response.StatusCode())
			case "completed":
				status, body := newProtocolAgent(
					t,
					team.App,
					agent,
				).reportVerification(verificationID, agentclient.VerificationReport{Status: agentclient.VerificationReportStatusCOMPLETED})
				require.Equal(t, http.StatusNoContent, status, string(body))
			}

			client := team.App.NewAgentClient(t, agent)
			stream, err := client.StreamVerificationBackupWithResponse(t.Context(), agent.Agent.ID, verificationID)
			require.NoError(t, err)
			requireAgentProblem(
				t,
				agentResponse{Status: stream.StatusCode(), Body: stream.Body},
				agentclient.Problem{Status: http.StatusGone, Code: agentclient.ProblemCodeVerificationGone},
			)

			for _, status := range []agentclient.VerificationReportStatus{agentclient.VerificationReportStatusCOMPLETED, agentclient.VerificationReportStatusFAILED} {
				report, err := client.ReportVerificationWithResponse(
					t.Context(),
					agent.Agent.ID,
					verificationID,
					agentclient.VerificationReport{Status: status},
				)
				require.NoError(t, err)
				requireAgentProblem(
					t,
					agentResponse{Status: report.StatusCode(), Body: report.Body},
					agentclient.Problem{Status: http.StatusGone, Code: agentclient.ProblemCodeVerificationGone},
				)
			}

			if state == "cancelled" {
				assert.Equal(t, apiclient.VerificationStatusCANCELED, team.read(t, verification.Id).Status)
			}
			if state == "reclaimed" {
				assert.Equal(t, apiclient.VerificationStatusRUNNING, team.read(t, verification.Id).Status)
				assert.NotEqual(t, agent.Agent.ID, *team.read(t, verification.Id).AgentId)
			}
		})
	}
}

func TestVerificationBackupStreamFailure(t *testing.T) {
	t.Parallel()

	t.Run("returns internal and preserves ownership when the stored backup file is missing", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		team.App.WaitForBackupRuns(t)
		team.App.RemoveLocalBackupFile(t, backup.Id)

		response, err := team.App.NewAgentClient(t, agent).
			StreamVerificationBackupWithResponse(t.Context(), agent.Agent.ID, verification.Id)
		require.NoError(t, err)
		requireAgentProblem(
			t,
			agentResponse{Status: response.StatusCode(), Body: response.Body},
			agentclient.Problem{Status: http.StatusInternalServerError, Code: agentclient.ProblemCodeInternal},
		)

		persistedVerification := team.read(t, verification.Id)
		assert.Equal(t, apiclient.VerificationStatusRUNNING, persistedVerification.Status)
		require.NotNil(t, persistedVerification.AgentId)
		assert.Equal(t, agent.Agent.ID, *persistedVerification.AgentId)
	})
}
