package verification_test

import (
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

const beyondAssignmentEligibility = 5*time.Minute + time.Microsecond

func (team verificationTeam) readAgent(t *testing.T, id uuid.UUID) apiclient.VerificationAgent {
	t.Helper()
	listed, err := team.Admin.Client.ListVerificationAgentsWithResponse(t.Context(), nil)
	require.NoError(t, err)
	require.NotNil(t, listed.JSON200, string(listed.Body))

	for _, listedAgent := range listed.JSON200.Items {
		if listedAgent.Id == id {
			return listedAgent
		}
	}

	require.FailNow(t, "verification agent is not listed", id.String())
	return apiclient.VerificationAgent{}
}

func requireClaimWithheld(
	t *testing.T,
	team verificationTeam,
	agent apptest.CreatedVerificationAgent,
	verification apiclient.Verification,
) {
	t.Helper()
	agentBeforeClaim := team.readAgent(t, agent.Agent.ID)

	status, body := newProtocolAgent(t, team.App, agent).claimVerification(claimBody(25))

	require.Equal(t, http.StatusNoContent, status, string(body))
	assert.Empty(t, body)
	withheldVerification := team.read(t, verification.Id)
	assert.Equal(t, apiclient.VerificationStatusPENDING, withheldVerification.Status)
	assert.Nil(t, withheldVerification.AgentId)
	assert.Equal(t, verification.AttemptCount, withheldVerification.AttemptCount)
	assert.Equal(t, agentBeforeClaim, team.readAgent(t, agent.Agent.ID))
}

func requireClaimAssigned(
	t *testing.T,
	team verificationTeam,
	agent apptest.CreatedVerificationAgent,
	verification apiclient.Verification,
) {
	t.Helper()
	status, body := newProtocolAgent(t, team.App, agent).claimVerification(claimBody(25))

	require.Equal(t, http.StatusOK, status, string(body))
	assignedVerification := team.read(t, verification.Id)
	assert.Equal(t, apiclient.VerificationStatusRUNNING, assignedVerification.Status)
	require.NotNil(t, assignedVerification.AgentId)
	assert.Equal(t, agent.Agent.ID, *assignedVerification.AgentId)
	assert.Equal(t, verification.AttemptCount, assignedVerification.AttemptCount)
}

func requireRejectedHeartbeat(t *testing.T, protocol protocolAgent, body string) {
	t.Helper()
	status, responseBody := protocol.sendRawHeartbeat(body)
	requireAgentProblem(
		t,
		agentResponse{Status: status, Body: responseBody},
		agentclient.Problem{Status: http.StatusBadRequest, Code: agentclient.ProblemCodeInvalidRequest},
	)
}

func TestVerificationClaimHeartbeat(t *testing.T) {
	t.Parallel()

	t.Run("withholds fitting work without an accepted heartbeat", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.start(t, team.backup(t).Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)

		requireClaimWithheld(t, team, agent, verification)
	})

	for _, scenario := range []struct {
		name       string
		age        time.Duration
		isEligible bool
	}{
		{name: "assigns fitting work when the accepted heartbeat is fresh", age: 5*time.Minute - time.Second, isEligible: true},
		{name: "assigns fitting work when the accepted heartbeat is exactly five minutes old", age: 5 * time.Minute, isEligible: true},
		{name: "withholds fitting work when the accepted heartbeat is just beyond five minutes old", age: beyondAssignmentEligibility, isEligible: false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			verification := team.start(t, team.backup(t).Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			// PostgreSQL keeps microseconds, so an unaligned clock would store a heartbeat older than it was sent.
			team.App.Clock.Set(team.App.Clock.Now().Truncate(time.Microsecond))
			newProtocolAgent(t, team.App, agent).requireAcceptedHeartbeat(nil)

			team.App.Clock.Advance(scenario.age)

			if scenario.isEligible {
				requireClaimAssigned(t, team, agent, verification)
			} else {
				requireClaimWithheld(t, team, agent, verification)
			}
		})
	}

	t.Run(
		"keeps withholding work without refreshing contact when an expired agent claims repeatedly",
		func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			verification := team.start(t, team.backup(t).Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			newProtocolAgent(t, team.App, agent).requireAcceptedHeartbeat(nil)
			team.App.Clock.Advance(beyondAssignmentEligibility)

			for range 3 {
				requireClaimWithheld(t, team, agent, verification)
			}
		},
	)
	t.Run("assigns work with the same credentials after a first accepted heartbeat", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.start(t, team.backup(t).Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		requireClaimWithheld(t, team, agent, verification)

		newProtocolAgent(t, team.App, agent).requireAcceptedHeartbeat(nil)

		requireClaimAssigned(t, team, agent, verification)
	})

	t.Run("assigns work with the same credentials after an expired agent heartbeats again", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.start(t, team.backup(t).Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		protocol := newProtocolAgent(t, team.App, agent)
		protocol.requireAcceptedHeartbeat(nil)
		team.App.Clock.Advance(beyondAssignmentEligibility)
		requireClaimWithheld(t, team, agent, verification)

		protocol.requireAcceptedHeartbeat(nil)

		requireClaimAssigned(t, team, agent, verification)
	})

	t.Run("assigns withheld work to a healthy agent after an expired agent is refused", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.start(t, team.backup(t).Id)
		expiredAgent := team.App.CreateVerificationAgent(t, team.Admin)
		newProtocolAgent(t, team.App, expiredAgent).requireAcceptedHeartbeat(nil)
		team.App.Clock.Advance(beyondAssignmentEligibility)
		requireClaimWithheld(t, team, expiredAgent, verification)

		healthyAgent := team.App.CreateVerificationAgent(t, team.Admin)
		newProtocolAgent(t, team.App, healthyAgent).requireAcceptedHeartbeat(nil)

		requireClaimAssigned(t, team, healthyAgent, verification)
	})

	t.Run("withholds work when only a gigabyte heartbeat was sent", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.start(t, team.backup(t).Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		agentBeforeRejection := team.readAgent(t, agent.Agent.ID)

		requireRejectedHeartbeat(t, newProtocolAgent(t, team.App, agent), gigabyteOnlyHeartbeat)

		assert.Equal(t, agentBeforeRejection, team.readAgent(t, agent.Agent.ID))
		assert.Nil(t, agentBeforeRejection.LastSeenAt)
		requireClaimWithheld(t, team, agent, verification)
	})

	t.Run("withholds work after expiry when rejected heartbeats followed the accepted one", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.start(t, team.backup(t).Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		protocol := newProtocolAgent(t, team.App, agent)
		protocol.requireAcceptedHeartbeat(nil)
		agentAfterAcceptance := team.readAgent(t, agent.Agent.ID)

		team.App.Clock.Advance(4 * time.Minute)
		requireRejectedHeartbeat(t, protocol, gigabyteOnlyHeartbeat)
		team.App.Clock.Advance(time.Minute + time.Microsecond)
		requireRejectedHeartbeat(t, protocol, missingMemoryHeartbeat)

		assert.Equal(t, agentAfterAcceptance, team.readAgent(t, agent.Agent.ID))
		requireClaimWithheld(t, team, agent, verification)
	})

	t.Run(
		"assigns work within the original window when a rejected heartbeat followed the accepted one",
		func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			verification := team.start(t, team.backup(t).Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			protocol := newProtocolAgent(t, team.App, agent)
			protocol.requireAcceptedHeartbeat(nil)
			agentAfterAcceptance := team.readAgent(t, agent.Agent.ID)

			team.App.Clock.Advance(time.Minute)
			requireRejectedHeartbeat(t, protocol, gigabyteOnlyHeartbeat)

			assert.Equal(t, agentAfterAcceptance, team.readAgent(t, agent.Agent.ID))
			requireClaimAssigned(t, team, agent, verification)
			assert.Equal(t, agentAfterAcceptance, team.readAgent(t, agent.Agent.ID))
		},
	)
}

func TestVerificationClaimHeartbeatPrecedence(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name      string
		isExpired bool
	}{
		{name: "when the agent has no accepted heartbeat", isExpired: false},
		{name: "when the agent heartbeat has expired", isExpired: true},
	} {
		newIneligibleAgent := func(t *testing.T) (*apptest.App, apptest.CreatedVerificationAgent) {
			t.Helper()
			app := apptest.New(t)
			agent := app.CreateVerificationAgent(t, app.CreateAdmin(t))

			if scenario.isExpired {
				newProtocolAgent(t, app, agent).requireAcceptedHeartbeat(nil)
				app.Clock.Advance(beyondAssignmentEligibility)
			}

			return app, agent
		}

		t.Run("rejects invalid claims "+scenario.name, func(t *testing.T) {
			t.Parallel()

			app, agent := newIneligibleAgent(t)
			protocol := newProtocolAgent(t, app, agent)
			invalidRequest := agentclient.Problem{
				Status: http.StatusBadRequest,
				Code:   agentclient.ProblemCodeInvalidRequest,
			}

			status, body := protocol.claimVerification(claimBody(-1))
			requireAgentProblem(t, agentResponse{Status: status, Body: body}, invalidRequest)

			status, body = protocol.sendRawClaim(`{"capacity":`)
			requireAgentProblem(t, agentResponse{Status: status, Body: body}, invalidRequest)
		})

		t.Run("refuses wrong credentials "+scenario.name, func(t *testing.T) {
			t.Parallel()

			app, agent := newIneligibleAgent(t)
			invalidCredentials := agent
			invalidCredentials.Token = "invalid"

			status, body := newProtocolAgent(t, app, invalidCredentials).claimVerification(claimBody(25))

			requireAgentProblem(
				t,
				agentResponse{Status: status, Body: body},
				agentclient.Problem{Status: http.StatusUnauthorized, Code: agentclient.ProblemCodeUnauthenticated},
			)
		})

		t.Run("limits repeated claims "+scenario.name, func(t *testing.T) {
			t.Parallel()

			app, agent := newIneligibleAgent(t)
			protocol := newProtocolAgent(t, app, agent)
			for range 10 {
				status, body := protocol.claimVerification(claimBody(25))
				require.Equal(t, http.StatusNoContent, status, string(body))
			}

			status, body := protocol.claimVerification(claimBody(25))

			requireAgentProblem(
				t,
				agentResponse{Status: status, Body: body},
				agentclient.Problem{Status: http.StatusTooManyRequests, Code: agentclient.ProblemCodeRateLimited},
			)
		})
	}
}

func TestVerificationAssignmentEligibilityScope(t *testing.T) {
	t.Parallel()

	t.Run("keeps ineligible agents listed and counted without a fresh heartbeat", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		neverSeenAgent := app.CreateVerificationAgent(t, admin)
		expiredAgent := app.CreateVerificationAgent(t, admin)
		newProtocolAgent(t, app, expiredAgent).requireAcceptedHeartbeat(nil)
		app.Clock.Advance(beyondAssignmentEligibility)

		listedAgents := app.ListVerificationAgents(t, admin)
		listedAgentIDs := make([]uuid.UUID, 0, len(listedAgents))
		for _, listedAgent := range listedAgents {
			listedAgentIDs = append(listedAgentIDs, listedAgent.ID)
		}
		assert.ElementsMatch(t, []uuid.UUID{neverSeenAgent.Agent.ID, expiredAgent.Agent.ID}, listedAgentIDs)

		count, err := admin.Client.GetVerificationAgentCountWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, count.JSON200, string(count.Body))
		assert.Equal(t, int64(2), count.JSON200.AgentCount)
	})

	t.Run("enqueues a verification when no agent is eligible", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		neverSeenAgent := team.App.CreateVerificationAgent(t, team.Admin)
		expiredAgent := team.App.CreateVerificationAgent(t, team.Admin)
		newProtocolAgent(t, team.App, expiredAgent).requireAcceptedHeartbeat(nil)
		team.App.Clock.Advance(beyondAssignmentEligibility)

		verification := team.start(t, team.backup(t).Id)

		assert.Equal(t, apiclient.VerificationStatusPENDING, verification.Status)
		requireClaimWithheld(t, team, neverSeenAgent, verification)
		requireClaimWithheld(t, team, expiredAgent, verification)
	})

	t.Run("lets an expired owner stream and report its job before the reaper runs", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		verification := team.start(t, team.backup(t).Id)
		team.App.WaitForBackupRuns(t)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		team.App.Clock.Advance(beyondAssignmentEligibility)

		stream, err := team.App.NewAgentClient(t, agent).
			StreamVerificationBackupWithResponse(t.Context(), agent.Agent.ID, verification.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, stream.StatusCode(), string(stream.Body))

		status, body := newProtocolAgent(t, team.App, agent).reportVerification(
			verification.Id,
			agentclient.VerificationReport{Status: agentclient.VerificationReportStatusCOMPLETED},
		)
		require.Equal(t, http.StatusNoContent, status, string(body))
		assert.Equal(t, apiclient.VerificationStatusCOMPLETED, team.read(t, verification.Id).Status)
	})
}
