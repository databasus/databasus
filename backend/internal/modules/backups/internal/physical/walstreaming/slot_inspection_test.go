package walstreaming

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestClassifySlotInspection(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		inspection      slotInspection
		expectedOutcome slotProbeOutcome
	}{
		{"reports an unreachable source when inspection fails", slotInspection{Error: errors.New("source unavailable")}, slotProbeOutcomeUnreachable},
		{"reports a missing slot when inspection finds none", slotInspection{}, slotProbeOutcomeSlotMissing},
		{"reports an unreachable source when its WAL position is unavailable", slotInspection{State: &pgphysical.Slot{}}, slotProbeOutcomeUnreachable},
		{"reports a reachable source when the slot has a WAL position", slotInspection{State: &pgphysical.Slot{CurrentLSN: new(walmath.LSN(42))}}, slotProbeOutcomeReachable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			probe := classifySlotInspection(testCase.inspection)

			assert.Equal(t, testCase.expectedOutcome, probe.Outcome)
			if probe.IsReachableWithSlot() {
				assert.Same(t, testCase.inspection.State, probe.State)
			} else {
				assert.Nil(t, probe.State)
			}
		})
	}
}
