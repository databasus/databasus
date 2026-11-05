package verification_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/agentclient"
	"databasus-backend/internal/lib/httpclienttest"
)

type protocolAgent struct {
	test        *testing.T
	app         *apptest.App
	credentials apptest.CreatedVerificationAgent
	httpClient  *http.Client
}

func newProtocolAgent(t *testing.T, app *apptest.App, credentials apptest.CreatedVerificationAgent) protocolAgent {
	t.Helper()
	return protocolAgent{test: t, app: app, credentials: credentials, httpClient: httpclienttest.New(t)}
}

func (a protocolAgent) recordHeartbeat(id uuid.UUID, body agentclient.HeartbeatRequest) (int, []byte) {
	a.test.Helper()
	response, err := a.app.NewAgentClient(a.test, a.credentials).RecordHeartbeatWithResponse(a.test.Context(), id, body)
	require.NoError(a.test, err)
	return response.StatusCode(), response.Body
}

func (a protocolAgent) requireAcceptedHeartbeat(ownedVerificationIDs []uuid.UUID) {
	a.test.Helper()
	status, body := a.recordHeartbeat(a.credentials.Agent.ID, heartbeatBody(ownedVerificationIDs))
	require.Equal(a.test, http.StatusOK, status, string(body))
}

func (a protocolAgent) claimVerification(body agentclient.ClaimRequest) (int, []byte) {
	a.test.Helper()
	response, err := a.app.NewAgentClient(a.test, a.credentials).
		ClaimVerificationWithResponse(a.test.Context(), a.credentials.Agent.ID, body)
	require.NoError(a.test, err)
	return response.StatusCode(), response.Body
}

func (a protocolAgent) reportVerification(id uuid.UUID, body agentclient.VerificationReport) (int, []byte) {
	a.test.Helper()
	response, err := a.app.NewAgentClient(a.test, a.credentials).
		ReportVerificationWithResponse(a.test.Context(), a.credentials.Agent.ID, id, body)
	require.NoError(a.test, err)
	return response.StatusCode(), response.Body
}

func (a protocolAgent) sendInvalidHeartbeat(id string) (int, []byte) {
	a.test.Helper()
	return a.postRawJSON("/api/v1/agent/verification/"+id+"/heartbeat", `{}`)
}

func (a protocolAgent) sendRawHeartbeat(body string) (int, []byte) {
	a.test.Helper()
	return a.postRawJSON("/api/v1/agent/verification/"+a.credentials.Agent.ID.String()+"/heartbeat", body)
}

func (a protocolAgent) sendRawClaim(body string) (int, []byte) {
	a.test.Helper()
	return a.postRawJSON("/api/v1/agent/verifications/"+a.credentials.Agent.ID.String()+"/claim", body)
}

func (a protocolAgent) postRawJSON(path, body string) (int, []byte) {
	a.test.Helper()
	request, err := http.NewRequestWithContext(
		a.test.Context(),
		http.MethodPost,
		a.app.URL+path,
		strings.NewReader(body),
	)
	require.NoError(a.test, err)

	request.Header.Set("Authorization", "Bearer "+a.credentials.Token)
	request.Header.Set("Content-Type", "application/json")

	response, err := a.httpClient.Do(request)
	require.NoError(a.test, err)
	defer func() { require.NoError(a.test, response.Body.Close()) }()
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(a.test, err)

	return response.StatusCode, responseBody
}

type agentResponse struct {
	Status int
	Body   []byte
}

func requireAgentProblem(
	t *testing.T,
	response agentResponse,
	expectedProblem agentclient.Problem,
) agentclient.Problem {
	t.Helper()
	require.Equal(t, expectedProblem.Status, response.Status, string(response.Body))
	var problem agentclient.Problem
	require.NoError(t, json.Unmarshal(response.Body, &problem))
	assert.Equal(t, expectedProblem.Status, problem.Status)
	assert.Equal(t, expectedProblem.Code, problem.Code)
	return problem
}
