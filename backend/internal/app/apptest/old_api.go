package apptest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// OldAPICall is a request to a route of the old code, which the contract does not describe yet.
type OldAPICall struct {
	Method string
	Path   string
	// Body is encoded as JSON, nil for none.
	Body any
}

// CallOldAPI sends the request with the session's token and returns the status and the body.
func (a *App) CallOldAPI(tb testing.TB, session Session, call OldAPICall) (int, []byte) {
	tb.Helper()

	return a.callOldAPI(tb, session.Token, call)
}

func (a *App) openOldAPI(tb testing.TB, token string, call OldAPICall) *http.Response {
	tb.Helper()

	var requestBody bytes.Buffer
	if call.Body != nil {
		err := json.NewEncoder(&requestBody).Encode(call.Body)
		if err != nil {
			tb.Fatalf("encode the body of %s %s: %v", call.Method, call.Path, err)
		}
	}

	request, err := http.NewRequestWithContext(tb.Context(), call.Method, a.URL+"/api/v1"+call.Path, &requestBody)
	if err != nil {
		tb.Fatalf("build %s %s: %v", call.Method, call.Path, err)
	}

	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		tb.Fatalf("send %s %s: %v", call.Method, call.Path, err)
	}

	return response
}

func (a *App) callOldAPI(tb testing.TB, token string, call OldAPICall) (int, []byte) {
	tb.Helper()
	response := a.openOldAPI(tb, token, call)

	defer func() { _ = response.Body.Close() }()

	var responseBody bytes.Buffer

	_, err := responseBody.ReadFrom(response.Body)
	if err != nil {
		tb.Fatalf("read the answer of %s %s: %v", call.Method, call.Path, err)
	}

	return response.StatusCode, responseBody.Bytes()
}
