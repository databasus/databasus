package contracttest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contract "databasus-backend/api"
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
			checkingDoer := &Doer{
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

const contractWithBinary = `
openapi: 3.0.3
info:
  title: Contract with an archive
  version: 1.0.0
servers:
  - url: /api/v1
paths:
  /archives:
    get:
      operationId: downloadArchive
      responses:
        "200":
          description: The archive stream
          headers:
            X-Archive-Id:
              required: true
              schema:
                type: string
                format: uuid
          content:
            application/x-tar:
              schema:
                type: string
                format: binary
`

func TestContractCheckingBinaryResponse(t *testing.T) {
	t.Parallel()
	t.Run("returns headers before the archive finishes when the contract declares a binary stream", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/x-tar")
			writer.Header().Set("X-Archive-Id", "0192f5c4-7c2a-7b3e-9d41-2f6a8c1e5b07")
			_, _ = writer.Write([]byte("archive"))
			assert.NoError(t, http.NewResponseController(writer).Flush())
			<-request.Context().Done()
		}))
		t.Cleanup(server.Close)

		router, err := httpserver.NewContractRouter([]byte(contractWithBinary))
		require.NoError(t, err)
		doer := &Doer{tb: t, contractRouter: router, httpClient: server.Client()}

		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/archives", nil)
		require.NoError(t, err)

		response, err := doer.Do(request)
		require.NoError(t, err)
		defer func() { _ = response.Body.Close() }()

		firstByte := make([]byte, 1)
		_, err = response.Body.Read(firstByte)
		require.NoError(t, err)
		assert.Equal(t, "a", string(firstByte))
	})
}

func TestContractCheckingBinaryHeaders(t *testing.T) {
	t.Parallel()
	t.Run("reports the invalid header when a binary response breaks its header schema", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/x-tar")
			writer.Header().Set("X-Archive-Id", "not-a-uuid")
			_, _ = writer.Write([]byte("archive"))
		}))
		t.Cleanup(server.Close)

		router, err := httpserver.NewContractRouter([]byte(contractWithBinary))
		require.NoError(t, err)
		recorder := &failureRecorder{TB: t, failures: nil, mutex: sync.Mutex{}}
		doer := &Doer{tb: recorder, contractRouter: router, httpClient: server.Client()}
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/v1/archives", nil)
		require.NoError(t, err)

		response, err := doer.Do(request)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())

		failures := recorder.getFailures()
		require.Len(t, failures, 1)
		assert.Contains(t, failures[0], "X-Archive-Id")
	})
}

func TestResponseContract(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		status              int
		contentType         string
		body                string
		expectedFailureText []string
	}{
		{
			name:                "passes when the response matches the contract",
			status:              http.StatusOK,
			contentType:         "application/json",
			body:                `{"version":"3.26.0"}`,
			expectedFailureText: nil,
		},
		{
			name:                "fails naming the operation, the status and the field when the version is a number",
			status:              http.StatusOK,
			contentType:         "application/json",
			body:                `{"version":326}`,
			expectedFailureText: []string{"getVersion", "status 200", `"/version"`},
		},
		{
			name:                "fails naming the status when an error body is not a Problem",
			status:              http.StatusInternalServerError,
			contentType:         "application/json",
			body:                `{"error":"something failed"}`,
			expectedFailureText: []string{"getVersion", "status 500"},
		},
		{
			name:                "fails naming the status when a Problem content type carries another body",
			status:              http.StatusInternalServerError,
			contentType:         "application/problem+json",
			body:                `{"error":"something failed"}`,
			expectedFailureText: []string{"getVersion", "status 500", "Problem"},
		},
		{
			name:                "fails when the operation answers a status the contract does not declare",
			status:              http.StatusCreated,
			contentType:         "application/problem+json",
			body:                `{"version":"3.26.0"}`,
			expectedFailureText: []string{"getVersion", "status 201", "Problem", `"code"`},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", testCase.contentType)
				writer.WriteHeader(testCase.status)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			t.Cleanup(server.Close)

			recordingTest := &failureRecorder{TB: t, failures: nil, mutex: sync.Mutex{}}
			router, err := httpserver.NewContractRouter(contract.OpenAPI)
			require.NoError(t, err)
			client := New(Deps{TB: recordingTest, ContractRouter: router, HTTPClient: server.Client()})

			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				server.URL+"/api/v1/system/version",
				nil,
			)
			require.NoError(t, err)
			response, err := client.Do(request)

			require.NoError(t, err)
			_ = response.Body.Close()
			assert.Equal(t, testCase.status, response.StatusCode)

			recordedFailures := recordingTest.getFailures()
			if testCase.expectedFailureText == nil {
				assert.Empty(t, recordedFailures)

				return
			}

			require.Len(t, recordedFailures, 1)

			for _, expectedText := range testCase.expectedFailureText {
				assert.Contains(t, recordedFailures[0], expectedText)
			}
		})
	}
}
