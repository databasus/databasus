//go:build testhelpers

package notifier_models

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

// PassthroughEncryptor lets provider tests assert on plaintext credentials without wiring the real
// key material of encryption.FieldEncryptor.
type PassthroughEncryptor struct{}

func (p PassthroughEncryptor) Encrypt(plaintext string) (string, error) {
	return plaintext, nil
}

func (p PassthroughEncryptor) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

type StubResponse struct {
	StatusCode int
	Body       string
}

type CapturedRequest struct {
	Method  string
	Path    string
	Query   url.Values
	Headers http.Header
	Body    string
}

type RequestRecorder struct {
	mu       sync.Mutex
	requests []CapturedRequest
}

func (r *RequestRecorder) GetRequestCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.requests)
}

func (r *RequestRecorder) GetLastRequest() CapturedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.requests[len(r.requests)-1]
}

func StartRecordingServer(t *testing.T, response StubResponse) (string, *RequestRecorder) {
	t.Helper()

	recorder := &RequestRecorder{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		recorder.add(CapturedRequest{
			Method:  request.Method,
			Path:    request.URL.Path,
			Query:   request.URL.Query(),
			Headers: request.Header.Clone(),
			Body:    string(body),
		})

		w.WriteHeader(response.StatusCode)

		if response.Body != "" {
			// Test server that replays a canned response body.
			// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
			_, _ = w.Write([]byte(response.Body))
		}
	}))
	t.Cleanup(server.Close)

	return server.URL, recorder
}

func (r *RequestRecorder) add(request CapturedRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = append(r.requests, request)
}
