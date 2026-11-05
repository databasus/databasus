package httpclienttest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// GetNetworkTimeoutError returns the error a real HTTP client reports when an endpoint accepts the
// connection and never answers. Storage providers surface this same error type, and it matches
// context.DeadlineExceeded, without waiting out a production timeout.
func GetNetworkTimeoutError(t *testing.T) error {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 50 * time.Millisecond}}
	defer client.CloseIdleConnections()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build timeout request: %v", err)
	}

	response, err := client.Do(request)
	if err == nil {
		_ = response.Body.Close()
		t.Fatal("expected the silent endpoint to time out")
	}

	return err
}
