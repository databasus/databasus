package agentclienttest

import (
	"context"
	"net/http"
	"testing"

	contract "databasus-backend/api"
	"databasus-backend/internal/lib/agentclient"
	"databasus-backend/internal/lib/contracttest"
	"databasus-backend/internal/lib/httpclienttest"
	"databasus-backend/internal/lib/httpserver"
)

func New(tb testing.TB, baseURL, token string) *agentclient.ClientWithResponses {
	tb.Helper()
	router, err := httpserver.NewContractRouter(contract.Agent)
	if err != nil {
		tb.Fatalf("load the agent contract: %v", err)
	}
	doer := contracttest.New(contracttest.Deps{TB: tb, ContractRouter: router, HTTPClient: httpclienttest.New(tb)})
	client, err := agentclient.NewClientWithResponses(baseURL+"/api/v1", agentclient.WithHTTPClient(doer),
		agentclient.WithRequestEditorFn(func(_ context.Context, request *http.Request) error {
			request.Header.Set("Authorization", "Bearer "+token)
			return nil
		}))
	if err != nil {
		tb.Fatalf("create the agent client: %v", err)
	}
	return client
}
