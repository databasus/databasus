package apptest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// StartDenyingS3 serves an S3 endpoint that denies every request. A denial is permanent, so an
// upload to it fails at once; an endpoint that cannot be reached is retried for minutes first.
func StartDenyingS3(t *testing.T) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/xml")
		response.WriteHeader(http.StatusForbidden)
		_, _ = response.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Storage unavailable</Message></Error>`))
	}))
	t.Cleanup(server.Close)

	return server.URL
}
