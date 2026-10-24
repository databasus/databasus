package apiclient

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
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
			client := New(recordingTest, server.URL)

			response, err := client.GetVersion(t.Context())

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
