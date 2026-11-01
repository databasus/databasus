package agentclienttest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name, body string
		hasFailure bool
	}{
		{"passes when the gone response matches the contract", `{"type":"about:blank","title":"Verification gone","status":410,"detail":"gone","code":"verification_gone"}`, false},
		{"fails naming operation and status when the gone body is not a Problem", `{"reason":"gone"}`, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "Bearer agent-token", request.Header.Get("Authorization"))
				writer.Header().Set("Content-Type", "application/problem+json")
				writer.WriteHeader(http.StatusGone)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			t.Cleanup(server.Close)

			recorder := &failureRecorder{TB: t}
			client := New(recorder, server.URL, "agent-token")
			response, err := client.StreamVerificationBackup(t.Context(), uuid.New(), uuid.New())
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())

			if !testCase.hasFailure {
				assert.Empty(t, recorder.failures)
				return
			}
			require.Len(t, recorder.failures, 1)
			assert.Contains(t, recorder.failures[0], "streamVerificationBackup")
			assert.Contains(t, recorder.failures[0], "status 410")
		})
	}
}

type failureRecorder struct {
	testing.TB
	failures []string
}

func (r *failureRecorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}
