package apptest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenOldPublicAPI(t *testing.T) {
	t.Run("leaves the response open when the caller needs to read it", func(t *testing.T) {
		testApp := New(t)
		response := testApp.OpenOldPublicAPI(t, OldAPICall{Method: http.MethodGet, Path: "/system/version"})
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.NoError(t, response.Body.Close())
	})
}
