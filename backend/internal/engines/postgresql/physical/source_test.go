package physical

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestProbeLiveness(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			server := startTestServer(t, image, containers.WithMaxWalSenders(1))

			// The probe must not take a WAL sender: on a source sized tightly for its replicas it would
			// fail during every backup and could starve the backup itself.
			t.Run("succeeds when every wal sender is taken", func(t *testing.T) {
				t.Parallel()

				_, err := server.openReplicationConn(t, getAdminRole())
				require.NoError(t, err)

				err = server.newSource(getAdminRole()).
					TestReplicationConnection(t.Context(), newTestLogger(t), getFullOnlyCheck())
				require.Error(t, err, "the only wal sender must be taken, or the probe proves nothing")

				assert.NoError(t, server.newSource(getAdminRole()).ProbeLiveness(t.Context(), newTestLogger(t)))
			})

			t.Run("returns error without the password when the password is wrong", func(t *testing.T) {
				t.Parallel()
				wrongPassword := "wrong-password-" + uuid.NewString()

				err := server.newSource(testRole{Name: getAdminRole().Name, Password: wrongPassword}).
					ProbeLiveness(t.Context(), newTestLogger(t))

				require.ErrorContains(t, err, "password authentication failed")
				assert.NotContains(t, err.Error(), wrongPassword)
			})
		})
	}
}

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestGetClusterSizeMb(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("returns a positive size when cluster is reachable", func(t *testing.T) {
			t.Parallel()

			clusterSizeMb, err := server.newSource(getAdminRole()).GetClusterSizeMb(t.Context(), newTestLogger(t))

			require.NoError(t, err)
			assert.Positive(t, clusterSizeMb)
		})
	})
}

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestOpenInspection(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("returns an ordinary connection that runs sql when cluster is reachable", func(t *testing.T) {
			t.Parallel()

			inspectionConn, err := server.newSource(getAdminRole()).OpenInspection(t.Context())
			require.NoError(t, err)

			t.Cleanup(func() {
				closeTestConn(t, inspectionConn.Close)
			})

			require.NoError(t, inspectionConn.Ping(t.Context()))
		})
	})
}
