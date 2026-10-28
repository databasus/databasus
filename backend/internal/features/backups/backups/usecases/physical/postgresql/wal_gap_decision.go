package usecases_physical_postgresql

import (
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

type walGapState struct {
	PreviousEnd    *walmath.LSN
	Claim          *physical_models.PhysicalWalSegment
	QueuedSegments []queuedWalSegment
	HasPredecessor bool
}

type walChainGap struct {
	Start walmath.LSN
	End   walmath.LSN
}

func decideWalChainGap(state walGapState) *walChainGap {
	if state.PreviousEnd == nil || *state.PreviousEnd == state.Claim.StartLSN || state.HasPredecessor ||
		isPredecessorQueued(state.QueuedSegments, state.Claim) {
		return nil
	}
	return &walChainGap{
		Start: max(*state.PreviousEnd, getHighestQueuedEndBelow(state.QueuedSegments, state.Claim)),
		End:   state.Claim.StartLSN,
	}
}
