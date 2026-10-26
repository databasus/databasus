package apptest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
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
	request.Header.Set("Authorization", "Bearer "+session.Token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		tb.Fatalf("send %s %s: %v", call.Method, call.Path, err)
	}

	defer func() { _ = response.Body.Close() }()

	var responseBody bytes.Buffer

	_, err = responseBody.ReadFrom(response.Body)
	if err != nil {
		tb.Fatalf("read the answer of %s %s: %v", call.Method, call.Path, err)
	}

	return response.StatusCode, responseBody.Bytes()
}

// CreateLocalStorage creates a local storage in the workspace through the old storages route and
// returns its ID.
func (a *App) CreateLocalStorage(tb testing.TB, session Session, workspaceID uuid.UUID) uuid.UUID {
	tb.Helper()

	status, body := a.CallOldAPI(tb, session, OldAPICall{
		Method: http.MethodPost,
		Path:   "/storages",
		Body: map[string]any{
			"workspaceId":  workspaceID,
			"type":         "LOCAL",
			"name":         "Storage " + uuid.NewString()[:8],
			"localStorage": map[string]any{},
		},
	})
	if status != http.StatusOK {
		tb.Fatalf("create the local storage answered %d: %s", status, body)
	}

	var createdStorage struct {
		ID uuid.UUID `json:"id"`
	}

	err := json.Unmarshal(body, &createdStorage)
	if err != nil {
		tb.Fatalf("decode the created storage: %v", err)
	}

	return createdStorage.ID
}
