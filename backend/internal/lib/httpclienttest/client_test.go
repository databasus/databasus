package httpclienttest_test

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/httpclienttest"
)

func TestClientConnectionIsolation(t *testing.T) {
	t.Parallel()

	t.Run("reuses its connection after another test server closes", func(t *testing.T) {
		t.Parallel()
		var connections atomic.Int64
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
		}))
		server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
			if state == http.StateNew {
				connections.Add(1)
			}
		}
		server.Start()
		t.Cleanup(server.Close)
		client := httpclienttest.New(t)

		for requestNumber := range 2 {
			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				server.URL,
				strings.NewReader("claim"),
			)
			require.NoError(t, err)
			response, err := client.Do(request)
			require.NoError(t, err)
			_, err = io.Copy(io.Discard, response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusNoContent, response.StatusCode)

			if requestNumber == 0 {
				unrelatedServer := httptest.NewServer(http.NotFoundHandler())
				unrelatedServer.Close()
			}
		}

		assert.Equal(t, int64(1), connections.Load())
	})
}
