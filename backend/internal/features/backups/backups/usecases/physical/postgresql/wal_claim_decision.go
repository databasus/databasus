package usecases_physical_postgresql

import physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"

type walClaimAction int

const (
	walClaimWait walClaimAction = iota
	walClaimRemoveLocal
	walClaimRecover
)

type existingWalClaim struct {
	Segment      *physical_models.PhysicalWalSegment
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
