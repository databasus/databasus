package cnpg_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestCNPGInterruptedBackupsPostgres17(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres17, getInterruptedBackupScenarios())
}

func TestCNPGInterruptedBackupsPostgres18(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres18, getInterruptedBackupScenarios())
}

func getInterruptedBackupScenarios() []cnpgScenario {
	var scenarios []cnpgScenario
	for _, interruption := range []struct {
		name string
		kind string
	}{
		{name: "replaces an interrupted full with ", kind: "FULL"},
		{name: "replaces an interrupted incremental with ", kind: "INCREMENTAL"},
	} {
		scenarios = append(scenarios, cnpgScenario{
			name: interruption.name,
			run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
				t.Helper()
				physicaltest.RunFailoverInterruptedBackup(t, physicaltest.FailoverInterruptionSpec{
					Manager: physicaltest.NewCNPGManager(spec),
					Kind:    interruption.kind,
				})
			},
		})
	}

	return scenarios
}
