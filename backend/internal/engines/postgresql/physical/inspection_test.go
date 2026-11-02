package physical

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestInspectionPing(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("answers and times out when an established source stops responding", func(t *testing.T) {
			t.Parallel()
			proxy, isStalled := startStallableSourceProxy(t, server.endpoint)
			proxyServer := testServer{endpoint: proxy, image: server.image, version: server.version}

			inspection := proxyServer.openInspection(t, getAdminRole())
			require.NoError(t, inspection.Ping(t.Context()))
			isStalled.Store(true)

			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			startedAt := time.Now()
			require.Error(t, inspection.Ping(ctx))
			require.Less(t, time.Since(startedAt), time.Second)
		})
	})
}
