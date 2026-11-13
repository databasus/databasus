package cnpg_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestCNPGRoutingOutagesPostgres17(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres17, getRoutingOutageScenarios())
}

func TestCNPGRoutingOutagesPostgres18(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres18, getRoutingOutageScenarios())
}

func getRoutingOutageScenarios() []cnpgScenario {
	var scenarios []cnpgScenario
	for _, kind := range []string{"FULL", "INCREMENTAL"} {
		scenarios = append(scenarios, cnpgScenario{
			name: "preserves the failure during a routing outage for " + kind + " with ",
			run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
				t.Helper()
				physicaltest.RunFailoverInterruptedBackup(t, physicaltest.FailoverInterruptionSpec{
					Manager:          physicaltest.NewCNPGManager(spec),
					Kind:             kind,
					ShouldHoldOutage: true,
				})
			},
		})
	}

	return scenarios
}
