package httpclienttest

import (
	"net/http"
	"testing"
)

func New(tb testing.TB) *http.Client {
	tb.Helper()

	defaultTransport, isHTTPTransport := http.DefaultTransport.(*http.Transport)
	if !isHTTPTransport {
		tb.Fatal("the default HTTP transport is not an *http.Transport")
	}

	// httptest.Server.Close closes the default transport's pool, including other tests' connections.
	transport := defaultTransport.Clone()
	tb.Cleanup(transport.CloseIdleConnections)

	return &http.Client{Transport: transport}
}
