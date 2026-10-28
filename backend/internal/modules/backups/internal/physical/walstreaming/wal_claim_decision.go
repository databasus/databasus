package walstreaming

import "databasus-backend/internal/modules/backups/internal/physical/catalog"

type walClaimAction int

const (
	walClaimWait walClaimAction = iota
	walClaimRemoveLocal
	walClaimRecover
)

type existingWalClaim struct {
	Segment      *catalog.WalSegment
	IsRecovering bool
}

func decideExistingWalClaim(claim existingWalClaim) walClaimAction {
	if claim.Segment == nil {
		return walClaimWait
	}
	if claim.Segment.FileName != nil {
		return walClaimRemoveLocal
	}
	if claim.IsRecovering {
		return walClaimRecover
	}
	return walClaimWait
}
