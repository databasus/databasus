//go:build testhelpers

package logicaltesting

import (
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
)

// The test has no other route to the bastioned database, so it reaches it the same way the backup
// does. That is the point of the topology rather than a limitation of it. The forwarder itself
// stays here and closes with the test; callers only need the address to dial.
func OpenForwardedEndpoint(t *testing.T, topology containers.BastionedDatabase) sshtunnel.Endpoint {
	t.Helper()

	forwarder, err := sshtunnel.Open(t.Context(), sshtunnel.OpenSpec{
		Config: bastion.GetTunnelConfig(topology),
		Target: sshtunnel.Endpoint{Host: topology.Database.Host, Port: topology.Database.Port},
		Logger: logger.GetLogger(),
	})
	require.NoError(t, err)
	t.Cleanup(forwarder.Close)

	return forwarder.GetLocalEndpoint()
}
