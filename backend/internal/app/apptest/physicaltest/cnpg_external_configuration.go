package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func RunCNPGExternalConfiguration(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	recovery := prepareCNPGIndependentRecovery(t, CNPGIndependentSpec{Manager: spec, Mode: CNPGRecoveryTarget})
	sourceConfiguration := recovery.Source.getServingConfiguration(t)

	recovery.boot(t)

	require.Equal(t, sourceConfiguration, recovery.Source.getServingConfiguration(t))
	recovery.Source.Cluster.GetIdentity(t)
}
