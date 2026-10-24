package apiclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"

	contract "databasus-backend/api"
	"databasus-backend/internal/lib/httpserver"
)

const apiBasePath = "/api/v1"

// kin-openapi checks only date and date-time by default, and its own uuid pattern accepts only UUID
// versions 1 to 5, so a UUIDv7 ID would fail. Any canonical UUID passes.
const uuidPattern = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`

var loadContractRouter = sync.OnceValues(func() (routers.Router, error) {
	return httpserver.NewContractRouter(contract.OpenAPI)
})

// New returns a client of the API served at baseURL, the address of the server without /api/v1.
// Every response that differs from the contract fails the test through tb.Errorf, and still reaches
// the test, so the output shows both the difference and the test's own assertions.
func New(tb testing.TB, baseURL string) *ClientWithResponses {
	tb.Helper()

	contractRouter, err := loadContractRouter()
	if err != nil {
		tb.Fatalf("load the API contract: %v", err)
	}

	checkingDoer := &contractCheckingDoer{tb: tb, contractRouter: contractRouter, httpClient: &http.Client{}}

	client, err := NewClientWithResponses(baseURL+apiBasePath, WithHTTPClient(checkingDoer))
	if err != nil {
		tb.Fatalf("create the API client: %v", err)
	}

	return client
}

type contractCheckingDoer struct {
	tb             testing.TB
	contractRouter routers.Router
	httpClient     *http.Client
}

func (d *contractCheckingDoer) Do(request *http.Request) (*http.Response, error) {
	response, err := d.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()

	if err != nil {
		return nil, fmt.Errorf("read the response of %s %s: %w", request.Method, request.URL.Path, err)
	}

	response.Body = io.NopCloser(bytes.NewReader(body))

	err = d.checkResponse(request.Context(), request, response, body)
	if err != nil {
		d.tb.Errorf("%v", err)
	}

	return response, nil
}

func (d *contractCheckingDoer) checkResponse(
	ctx context.Context,
	request *http.Request,
	response *http.Response,
	body []byte,
) error {
	route, pathParams, err := d.contractRouter.FindRoute(request)
	if err != nil {
		return fmt.Errorf("%s %s is not an operation of the contract: %w", request.Method, request.URL.Path, err)
	}

	err = openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    request,
			PathParams: pathParams,
			Route:      route,
		},
		Status: response.StatusCode,
		Header: response.Header,
		Body:   io.NopCloser(bytes.NewReader(body)),
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true,
			MultiError:            true,
			SchemaValidationOptions: []openapi3.SchemaValidationOption{
				openapi3.WithStringFormatValidator("uuid", openapi3.NewRegexpFormatValidator(uuidPattern)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("the response of %s with status %d breaks the contract: %w",
			route.Operation.OperationID, response.StatusCode, err)
	}

	return nil
}
