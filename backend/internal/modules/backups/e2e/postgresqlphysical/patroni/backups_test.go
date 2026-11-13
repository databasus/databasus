package patroni_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

const (
	postgres17Image = "postgres:17"
	postgres18Image = "postgres:18"
)

type patroniScenario struct {
	name string
	run  func(*testing.T, string)
}

func TestPatroniRecoveryPostgres17(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres17Image, getRecoveryScenarios())
}

func TestPatroniRecoveryPostgres18(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres18Image, getRecoveryScenarios())
}

func TestPatroniIndependentRecoveryPostgres17(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres17Image, getIndependentRecoveryScenarios())
}

func TestPatroniIndependentRecoveryPostgres18(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres18Image, getIndependentRecoveryScenarios())
}

func TestPatroniSlotsPostgres17(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres17Image, getSlotScenarios())
}

func TestPatroniSlotsPostgres18(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres18Image, getSlotScenarios())
}

func TestPatroniInterruptedBackupsPostgres17(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres17Image, getInterruptedBackupScenarios())
}

func TestPatroniInterruptedBackupsPostgres18(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres18Image, getInterruptedBackupScenarios())
}

func TestPatroniRoutingOutagesPostgres17(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres17Image, getRoutingOutageScenarios())
}

func TestPatroniRoutingOutagesPostgres18(t *testing.T) {
	t.Parallel()
	runPatroniScenarios(t, postgres18Image, getRoutingOutageScenarios())
}

func runPatroniScenarios(t *testing.T, image string, scenarios []patroniScenario) {
	t.Helper()

	for _, scenario := range scenarios {
		t.Run(scenario.name+image, func(t *testing.T) {
			t.Parallel()
			scenario.run(t, image)
		})
	}
}

func getRecoveryScenarios() []patroniScenario {
	return []patroniScenario{
		{name: "restores initial schema and sequences with ", run: func(t *testing.T, image string) {
			t.Helper()
			physicaltest.RunFailoverInitialRecovery(t, physicaltest.NewPatroniManager(image))
		}},
		{name: "recovers after two switchovers with ", run: func(t *testing.T, image string) {
			t.Helper()
			physicaltest.RunFailoverSwitchovers(t, physicaltest.NewPatroniManager(image))
		}},
		{name: "recovers after primary loss with ", run: func(t *testing.T, image string) {
			t.Helper()
			physicaltest.RunFailoverPrimaryLoss(t, physicaltest.NewPatroniManager(image))
		}},
	}
}

func getIndependentRecoveryScenarios() []patroniScenario {
	scenarios := []patroniScenario{
		{
			name: "uses referenced authentication when root defaults coexist with ",
			run: func(t *testing.T, image string) {
				t.Helper()

				physicaltest.RunPatroniIndependentRecovery(t, physicaltest.PatroniIndependentSpec{
					Image:                           image,
					Mode:                            "backup",
					ShouldUseExternalConfig:         false,
					ShouldHaveDefaultAuthentication: true,
				})
			},
		},
	}

	for _, mode := range []string{"target", "latest", "backup", "external-configuration"} {
		scenarios = append(
			scenarios,
			patroniScenario{
				name: "recovers independently with " + mode + " for ",
				run: func(t *testing.T, image string) {
					t.Helper()
					recoveryMode := mode
					if mode == "external-configuration" {
						recoveryMode = "latest"
					}

					physicaltest.RunPatroniIndependentRecovery(
						t,
						physicaltest.PatroniIndependentSpec{
							Image:                           image,
							Mode:                            recoveryMode,
							ShouldUseExternalConfig:         mode == "external-configuration",
							ShouldHaveDefaultAuthentication: false,
						},
					)
				},
			},
		)
	}

	return scenarios
}

func getSlotScenarios() []patroniScenario {
	var scenarios []patroniScenario

	for _, slotMode := range []struct {
		name string
		mode string
	}{
		{name: "keeps the ignored slot with ", mode: "protected"},
		{name: "handles an absent slot after promotion with ", mode: "absent"},
		{name: "detects removal of an unprotected slot with ", mode: "unprotected"},
		{name: "refuses a target after slot loss with ", mode: "lost"},
	} {
		scenarios = append(scenarios, patroniScenario{name: slotMode.name, run: func(t *testing.T, image string) {
			t.Helper()
			physicaltest.RunPatroniSlot(t, physicaltest.PatroniSlotSpec{Image: image, Mode: slotMode.mode})
		}})
	}

	return scenarios
}

func getInterruptedBackupScenarios() []patroniScenario {
	var scenarios []patroniScenario

	for _, interruption := range []struct {
		name string
		kind string
	}{
		{name: "replaces an interrupted full with ", kind: "FULL"},
		{name: "replaces an interrupted incremental with ", kind: "INCREMENTAL"},
	} {
		scenarios = append(scenarios, patroniScenario{name: interruption.name, run: func(t *testing.T, image string) {
			t.Helper()
			physicaltest.RunFailoverInterruptedBackup(
				t,
				physicaltest.FailoverInterruptionSpec{
					Manager: physicaltest.NewPatroniManager(image),
					Kind:    interruption.kind,
				},
			)
		}})
	}

	return scenarios
}

func getRoutingOutageScenarios() []patroniScenario {
	var scenarios []patroniScenario

	for _, kind := range []string{"FULL", "INCREMENTAL"} {
		scenarios = append(
			scenarios,
			patroniScenario{
				name: "preserves the failure during a routing outage for " + kind + " with ",
				run: func(t *testing.T, image string) {
					t.Helper()
					physicaltest.RunFailoverInterruptedBackup(
						t,
						physicaltest.FailoverInterruptionSpec{
							Manager:          physicaltest.NewPatroniManager(image),
							Kind:             kind,
							ShouldHoldOutage: true,
						},
					)
				},
			},
		)
	}

	return scenarios
}
