package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckServerAlive(t *testing.T) {
	t.Parallel()

	t.Run("returns no error when the server answers 200 on the version path", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "/api/v1/system/version", request.URL.Path)
			writer.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		require.NoError(t, checkServerAlive(t.Context(), server.URL))
	})

	t.Run("returns an error with the status when the server answers another status", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)

		err := checkServerAlive(t.Context(), server.URL)

		require.ErrorIs(t, err, ErrUnexpectedStatus)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("returns an error when the server is unreachable", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		unreachableURL := server.URL
		server.Close()

		require.Error(t, checkServerAlive(t.Context(), unreachableURL))
	})
}
