package cnpg_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestCNPGRecoveryPostgres17(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres17, getRecoveryScenarios())
}

func TestCNPGRecoveryPostgres18(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres18, getRecoveryScenarios())
}

func getRecoveryScenarios() []cnpgScenario {
	return []cnpgScenario{
		{
			name: "restores initial schema and sequences with ",
			run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
				t.Helper()
				physicaltest.RunFailoverInitialRecovery(t, physicaltest.NewCNPGManager(spec))
			},
		},
		{name: "recovers after two switchovers with ", run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
			t.Helper()
			physicaltest.RunFailoverSwitchovers(t, physicaltest.NewCNPGManager(spec))
		}},
		{name: "recovers after primary loss with ", run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
			t.Helper()
			physicaltest.RunFailoverPrimaryLoss(t, physicaltest.NewCNPGManager(spec))
		}},
	}
}
