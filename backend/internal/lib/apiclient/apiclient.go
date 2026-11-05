package apiclient

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/routers"

	contract "databasus-backend/api"
	"databasus-backend/internal/lib/contracttest"
	"databasus-backend/internal/lib/httpclienttest"
	"databasus-backend/internal/lib/httpserver"
)

const apiBasePath = "/api/v1"

//nolint:gochecknoglobals // The immutable contract router is parsed once and shared by test clients; it holds no application state.
var loadContractRouter = sync.OnceValues(func() (routers.Router, error) {
	return httpserver.NewContractRouter(contract.OpenAPI)
})

// New returns a client of the API served at baseURL, the address of the server without /api/v1.
// Every response that differs from the contract fails the test through tb.Errorf, and still reaches
// the test, so the output shows both the difference and the test's own assertions. Options such as
// WithSessionToken apply to every request.
func New(tb testing.TB, baseURL string, options ...ClientOption) *ClientWithResponses {
	tb.Helper()

	contractRouter, err := loadContractRouter()
	if err != nil {
		tb.Fatalf("load the API contract: %v", err)
	}

	checkingDoer := contracttest.New(
		contracttest.Deps{TB: tb, ContractRouter: contractRouter, HTTPClient: httpclienttest.New(tb)},
	)

	client, err := NewClientWithResponses(baseURL+apiBasePath, append([]ClientOption{WithHTTPClient(checkingDoer)},
		options...)...)
	if err != nil {
		tb.Fatalf("create the API client: %v", err)
	}

	return client
}

// WithSessionToken signs every request in with the session token a sign-in operation returned.
func WithSessionToken(token string) ClientOption {
	return WithRequestEditorFn(func(_ context.Context, request *http.Request) error {
		request.Header.Set("Authorization", "Bearer "+token)

		return nil
	})
}
