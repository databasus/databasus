// Package providertest holds what the tests of the notifier providers share: an HTTP endpoint that
// records what a provider sent, a field encryptor with a key of its own and a check of a refused
// setting.
package providertest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

type StubResponse struct {
	StatusCode int
	Body       string
}

type CapturedRequest struct {
	Method  string
	Host    string
	Path    string
	Query   url.Values
	Headers http.Header
	Body    string
}

type RequestRecorder struct {
	mutex    sync.Mutex
	requests []CapturedRequest
}

// StartRecordingServer answers every request with response and records it.
func StartRecordingServer(tb testing.TB, response StubResponse) (string, *RequestRecorder) {
	tb.Helper()

	recorder := &RequestRecorder{mutex: sync.Mutex{}, requests: nil}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		recorder.add(CapturedRequest{
			Method:  request.Method,
			Host:    request.Host,
			Path:    request.URL.Path,
			Query:   request.URL.Query(),
			Headers: request.Header.Clone(),
			Body:    string(body),
		})

		writer.WriteHeader(response.StatusCode)

		if response.Body != "" {
			// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
			_, _ = writer.Write([]byte(response.Body))
		}
	}))
	tb.Cleanup(server.Close)

	return server.URL, recorder
}

func (r *RequestRecorder) GetRequestCount() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return len(r.requests)
}

func (r *RequestRecorder) GetLastRequest() CapturedRequest {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.requests[len(r.requests)-1]
}

func (r *RequestRecorder) add(request CapturedRequest) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.requests = append(r.requests, request)
}
