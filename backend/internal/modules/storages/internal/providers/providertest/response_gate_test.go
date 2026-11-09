package providertest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestResponseGate(t *testing.T) {
	t.Parallel()

	t.Run("delivers the original response when the gate is released", func(t *testing.T) {
		t.Parallel()
		forwarder, client := startResponseGateway(t)
		gate := forwarder.PauseResponses(t)
		responses := make(chan []byte, 1)
		requestErrors := make(chan error, 1)
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
			fmt.Sprintf("http://%s:%d", forwarder.Host, forwarder.Port), nil)
		require.NoError(t, err)

		go func() {
			response, requestError := client.Do(request)
			var content []byte
			if response != nil {
				defer func() { _ = response.Body.Close() }()
				content, requestError = io.ReadAll(response.Body)
			}

			responses <- content
			requestErrors <- requestError
		}()

		gate.WaitUntilBlocked(t)

		select {
		case <-responses:
			t.Fatal("a storage response passed the closed gate")
		default:
		}

		gate.Release()
		testwait.Until(t, "the released HTTP request to finish", func(context.Context) error {
			select {
			case requestError := <-requestErrors:
				require.NoError(t, requestError)
				return nil
			default:
				return fmt.Errorf("the HTTP request is still running")
			}
		})

		assert.Equal(t, "storage response", string(<-responses))
		assert.Equal(t, 1, forwarder.GetAcceptedCount())
	})

	t.Run("observes the closed client connection before a withheld response is released", func(t *testing.T) {
		t.Parallel()
		forwarder, client := startResponseGateway(t)
		gate := forwarder.PauseResponses(t)
		requestCtx, cancel := context.WithCancel(t.Context())
		defer cancel()
		requestErrors := make(chan error, 1)
		request, err := http.NewRequestWithContext(requestCtx, http.MethodGet,
			fmt.Sprintf("http://%s:%d", forwarder.Host, forwarder.Port), nil)
		require.NoError(t, err)

		go func() {
			response, requestError := client.Do(request)
			if response != nil {
				_ = response.Body.Close()
			}
			requestErrors <- requestError
		}()

		gate.WaitUntilBlocked(t)
		cancel()
		testwait.Until(t, "the cancelled HTTP request to finish", func(context.Context) error {
			select {
			case requestError := <-requestErrors:
				require.ErrorIs(t, requestError, context.Canceled)
				return nil
			default:
				return fmt.Errorf("the HTTP request is still running")
			}
		})

		testwait.Until(t, "the gateway to observe client disconnection", func(context.Context) error {
			if forwarder.GetClosedCount() != 1 {
				return fmt.Errorf("%d client connections closed", forwarder.GetClosedCount())
			}

			return nil
		})

		gate.Release()
	})
}

func startResponseGateway(t *testing.T) (*CountingForwarder, *http.Client) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "storage response")
	}))
	t.Cleanup(server.Close)
	forwarder := StartCountingForwarder(t, strings.TrimPrefix(server.URL, "http://"))

	return forwarder, server.Client()
}
