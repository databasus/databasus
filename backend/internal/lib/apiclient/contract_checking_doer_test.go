package apiclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/httpserver"
)

const contractWithUUID = `
openapi: 3.0.3
info:
  title: Contract with a UUID field
  version: 1.0.0
servers:
  - url: /api/v1
paths:
  /storages/current:
    get:
      operationId: getCurrentStorage
      responses:
        "200":
          description: The current storage
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id:
                    type: string
                    format: uuid
`

func TestContractCheckingDoer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		body          string
		isFailureSeen bool
	}{
		{
			name:          "passes when an ID is a UUIDv7",
			body:          `{"id":"0192f5c4-7c2a-7b3e-9d41-2f6a8c1e5b07"}`,
			isFailureSeen: false,
		},
		{name: "fails when an ID is not a UUID", body: `{"id":"not-a-uuid"}`, isFailureSeen: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(testCase.body))
			}))
			t.Cleanup(server.Close)

			contractRouter, err := httpserver.NewContractRouter([]byte(contractWithUUID))
			require.NoError(t, err)

			recordingTest := &failureRecorder{TB: t, failures: nil, mutex: sync.Mutex{}}
			checkingDoer := &contractCheckingDoer{
				tb:             recordingTest,
				contractRouter: contractRouter,
				httpClient:     &http.Client{},
			}

			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				server.URL+"/api/v1/storages/current",
				nil,
			)
			require.NoError(t, err)

			response, err := checkingDoer.Do(request)
			require.NoError(t, err)
			_ = response.Body.Close()

			recordedFailures := recordingTest.getFailures()
			if !testCase.isFailureSeen {
				assert.Empty(t, recordedFailures)

				return
			}

			require.Len(t, recordedFailures, 1)
			assert.Contains(t, recordedFailures[0], "getCurrentStorage")
			assert.Contains(t, recordedFailures[0], `"/id"`)
		})
	}
}

// failureRecorder keeps the failures the doer reports instead of failing the surrounding test,
// which checks them.
type failureRecorder struct {
	testing.TB

	failures []string
	mutex    sync.Mutex
}

func (r *failureRecorder) Errorf(format string, args ...any) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *failureRecorder) getFailures() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return append([]string(nil), r.failures...)
}
