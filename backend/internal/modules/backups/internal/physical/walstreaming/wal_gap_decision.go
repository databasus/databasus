package walstreaming

import (
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type walGapState struct {
	PreviousEnd    *walmath.LSN
	Claim          *catalog.WalSegment
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
