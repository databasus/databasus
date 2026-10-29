package apiclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSessionToken(t *testing.T) {
	t.Parallel()

	t.Run("sends the token as a bearer authorization header", func(t *testing.T) {
		t.Parallel()
		authorizationByRequest := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authorizationByRequest <- request.Header.Get("Authorization")

			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"version":"3.26.0"}`))
		}))
		t.Cleanup(server.Close)

		client := New(t, server.URL, WithSessionToken("session-token"))

		response, err := client.GetVersion(t.Context())

		require.NoError(t, err)
		_ = response.Body.Close()
		assert.Equal(t, "Bearer session-token", <-authorizationByRequest)
	})
}
