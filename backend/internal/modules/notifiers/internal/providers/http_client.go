package providers

import (
	"net/http"
	"time"
)

// sendTimeout bounds one request of a provider even when the caller's context has no deadline.
const sendTimeout = 30 * time.Second

// NewHTTPClient returns a client with the send timeout. A nil transport uses a copy of the default
// one, which takes the proxy from the environment. The copy keeps its own connection pool, because
// closing the idle connections of the shared default transport also breaks a connection that a
// send has just taken from it.
func NewHTTPClient(transport http.RoundTripper) *http.Client {
	if transport == nil {
		transport = newDefaultTransport()
	}

	return &http.Client{Transport: transport, Timeout: sendTimeout, CheckRedirect: nil, Jar: nil}
}

func newDefaultTransport() http.RoundTripper {
	defaultTransport, isHTTPTransport := http.DefaultTransport.(*http.Transport)
	if !isHTTPTransport {
		return http.DefaultTransport
	}

	return defaultTransport.Clone()
}
