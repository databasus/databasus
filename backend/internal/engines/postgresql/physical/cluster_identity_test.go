package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/testing/containers"
)

const defaultWalSegmentSizeBytes = 16 * 1024 * 1024

func TestInspectCluster(t *testing.T) {
	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("returns version, system identifier and wal segment size when cluster is reachable", func(t *testing.T) {
			t.Parallel()

			clusterIdentity, err := server.newSource(adminRole).InspectCluster(t.Context(), newTestLogger(t))

			require.NoError(t, err)
			assert.Equal(t, server.version, clusterIdentity.Version)
			assert.Equal(t, server.readSystemIdentifier(t), clusterIdentity.SystemIdentifier)
			assert.Equal(t, int64(defaultWalSegmentSizeBytes), clusterIdentity.WalSegmentSizeBytes)
		})

		t.Run("returns error with wrong password", func(t *testing.T) {
			t.Parallel()

			wrongPasswordRole := testRole{Name: adminRole.Name, Password: adminRole.Password + "-wrong"}
			_, err := server.newSource(wrongPasswordRole).InspectCluster(t.Context(), newTestLogger(t))

			assert.ErrorContains(t, err, "password authentication failed")
		})
	})

	t.Run("returns unsupported version error when cluster runs postgres 16", func(t *testing.T) {
		server := testServer{endpoint: containers.StartPostgres(t, "postgres:16"), image: "postgres:16", version: ""}

		_, err := server.newSource(adminRole).InspectCluster(t.Context(), newTestLogger(t))

		require.ErrorIs(t, err, ErrUnsupportedVersion)
	})
}
