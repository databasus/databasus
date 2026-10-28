package apptest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
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

func (a *App) CallOldPublicAPI(tb testing.TB, call OldAPICall) (int, []byte) {
	tb.Helper()

	return a.callOldAPI(tb, "", call)
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

func (a *App) DownloadOldPublicAPI(tb testing.TB, path, destinationPath string) int {
	tb.Helper()

	request, err := http.NewRequestWithContext(tb.Context(), http.MethodGet, a.URL+"/api/v1"+path, nil)
	if err != nil {
		tb.Fatalf("build download request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		tb.Fatalf("download the old public API response: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		tb.Fatalf("create download destination: %v", err)
	}
	defer func() { _ = destination.Close() }()

	if _, err := io.Copy(destination, response.Body); err != nil {
		tb.Fatalf("write download destination: %v", err)
	}
	if err := destination.Close(); err != nil {
		tb.Fatalf("close download destination: %v", err)
	}

	return response.StatusCode
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

func (a *App) OpenOldPublicAPI(tb testing.TB, call OldAPICall) *http.Response {
	tb.Helper()
	response := a.openOldAPI(tb, "", call)
	tb.Cleanup(func() { _ = response.Body.Close() })
	return response
}
