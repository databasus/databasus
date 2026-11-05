package contracttest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
)

// kin-openapi checks only date and date-time by default, and its own uuid pattern accepts only UUID
// versions 1 to 5, so a UUIDv7 ID would fail. Any canonical UUID passes.
const uuidPattern = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`

type Doer struct {
	tb             testing.TB
	contractRouter routers.Router
	httpClient     *http.Client
}

type Deps struct {
	TB             testing.TB
	ContractRouter routers.Router
	HTTPClient     *http.Client
}

func New(deps Deps) *Doer {
	return &Doer{tb: deps.TB, contractRouter: deps.ContractRouter, httpClient: deps.HTTPClient}
}

func (d *Doer) Do(request *http.Request) (*http.Response, error) {
	response, err := d.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if d.isBinaryResponse(request, response) {
		if err := d.checkResponse(request.Context(), request, response, nil); err != nil {
			d.tb.Errorf("%v", err)
		}
		return response, nil
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

func (d *Doer) checkResponse(
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
			ExcludeResponseBody:   d.isBinaryResponse(request, response),
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

func (d *Doer) isBinaryResponse(request *http.Request, response *http.Response) bool {
	route, _, err := d.contractRouter.FindRoute(request)
	if err != nil {
		return false
	}
	responseSchema := route.Operation.Responses.Status(response.StatusCode)
	if responseSchema == nil {
		responseSchema = route.Operation.Responses.Default()
	}
	if responseSchema == nil || responseSchema.Value == nil {
		return false
	}
	media := responseSchema.Value.Content.Get(response.Header.Get("Content-Type"))
	return media != nil && media.Schema != nil && media.Schema.Value != nil && media.Schema.Value.Format == "binary"
}
