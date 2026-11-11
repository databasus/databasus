package patroni_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

type patroniScenario struct {
	name string
	run  func(*testing.T, string)
}

func TestPatroniPhysicalBackups(t *testing.T) {
	t.Parallel()

	capacity := make(chan struct{}, 2)
	scenarios := []patroniScenario{
		{name: "restores initial schema and sequences with ", run: physicaltest.RunPatroniInitialRecovery},
		{name: "recovers after two switchovers with ", run: physicaltest.RunPatroniSwitchovers},
		{name: "recovers after primary loss with ", run: physicaltest.RunPatroniPrimaryLoss},
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

	for _, kind := range []string{"FULL", "INCREMENTAL"} {
		name := "replaces an interrupted full with "

		if kind == "INCREMENTAL" {
			name = "replaces an interrupted incremental with "
		}

		scenarios = append(scenarios, patroniScenario{name: name, run: func(t *testing.T, image string) {
			t.Helper()
			physicaltest.RunPatroniInterruptedBackup(t, physicaltest.PatroniInterruptionSpec{Image: image, Kind: kind})
		}})

		scenarios = append(
			scenarios,
			patroniScenario{
				name: "preserves the failure during a routing outage for " + kind + " with ",
				run: func(t *testing.T, image string) {
					t.Helper()
					physicaltest.RunPatroniInterruptedBackup(
						t,
						physicaltest.PatroniInterruptionSpec{Image: image, Kind: kind, ShouldHoldOutage: true},
					)
				},
			},
		)
	}

	for _, image := range []string{"postgres:17", "postgres:18"} {
		for _, scenario := range scenarios {
			t.Run(scenario.name+image, func(t *testing.T) {
				t.Parallel()

				select {
				case capacity <- struct{}{}:
				case <-t.Context().Done():
					t.Fatal("canceled waiting for Patroni fixture capacity")
				}
				t.Cleanup(func() { <-capacity })
				scenario.run(t, image)
			})
		}
	}
}
