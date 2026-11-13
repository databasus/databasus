package cnpg_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestCNPGSlotsPostgres17(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres17, getSlotScenarios())
}

func TestCNPGSlotsPostgres18(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres18, getSlotScenarios())
}

func getSlotScenarios() []cnpgScenario {
	return []cnpgScenario{
		{name: "keeps the synchronized slot with ", run: physicaltest.RunCNPGSynchronizedSlot},
		{name: "handles an excluded slot after promotion with ", run: physicaltest.RunCNPGExcludedSlot},
		{
			name: "detects removal after synchronization is turned off with ",
			run:  physicaltest.RunCNPGUnsynchronizedSlot,
		},
		{name: "refuses a target after slot loss with ", run: physicaltest.RunCNPGLostSlot},
		{
			name: "refuses a target across a missing WAL interval on one timeline with ",
			run:  physicaltest.RunCNPGLostSlotGap,
		},
	}
}
