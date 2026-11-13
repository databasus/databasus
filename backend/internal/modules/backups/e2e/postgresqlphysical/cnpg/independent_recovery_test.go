package cnpg_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestCNPGIndependentRecoveryPostgres17(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres17, getIndependentRecoveryScenarios())
}

func TestCNPGIndependentRecoveryPostgres18(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres18, getIndependentRecoveryScenarios())
}

func getIndependentRecoveryScenarios() []cnpgScenario {
	var scenarios []cnpgScenario
	for _, mode := range []physicaltest.CNPGRecoveryMode{
		physicaltest.CNPGRecoveryTarget,
		physicaltest.CNPGRecoveryLatest,
		physicaltest.CNPGRecoveryBackup,
	} {
		scenarios = append(scenarios, cnpgScenario{
			name: "recovers independently with " + string(mode) + " for ",
			run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
				t.Helper()
				physicaltest.RunCNPGIndependentRecovery(t, physicaltest.CNPGIndependentSpec{Manager: spec, Mode: mode})
			},
		})
	}

	return append(
		scenarios,
		cnpgScenario{
			name: "boots with external configuration overridden as instructed for ",
			run:  physicaltest.RunCNPGExternalConfiguration,
		},
		cnpgScenario{
			name: "enforces the authentication rules copied in the data directory for ",
			run:  physicaltest.RunCNPGCopiedAuthentication,
		},
	)
}
