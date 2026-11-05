package providers

import (
	"net/http"
	"time"
)

// sendTimeout bounds one request of a provider even when the caller's context has no deadline.
const sendTimeout = 30 * time.Second

// NewHTTPClient returns a client with the send timeout. A nil transport uses the default one, which
// takes the proxy from the environment.
func NewHTTPClient(transport http.RoundTripper) *http.Client {
	return &http.Client{Transport: transport, Timeout: sendTimeout, CheckRedirect: nil, Jar: nil}
}
