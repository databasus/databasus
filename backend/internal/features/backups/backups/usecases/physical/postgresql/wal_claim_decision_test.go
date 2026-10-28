package usecases_physical_postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

func TestDecideExistingWalClaim(t *testing.T) {
	for _, testCase := range []struct {
		name           string
		claim          existingWalClaim
		expectedAction walClaimAction
	}{
		{"keeps the local file when a conflicting row disappeared", existingWalClaim{}, walClaimWait},
		{"keeps the local file during recovery when its row disappeared", existingWalClaim{IsRecovering: true}, walClaimWait},
		{"removes the local file when the segment is already committed", existingWalClaim{Segment: &physical_models.PhysicalWalSegment{FileName: new("archived")}}, walClaimRemoveLocal},
		{"removes a recovered local file when its segment is already committed", existingWalClaim{Segment: &physical_models.PhysicalWalSegment{FileName: new("archived")}, IsRecovering: true}, walClaimRemoveLocal},
		{"keeps the local file when an active claim is uncommitted", existingWalClaim{Segment: &physical_models.PhysicalWalSegment{}}, walClaimWait},
		{"takes over an uncommitted claim when startup recovers the segment", existingWalClaim{Segment: &physical_models.PhysicalWalSegment{}, IsRecovering: true}, walClaimRecover},
	} {
		t.Run(
			testCase.name,
			func(t *testing.T) { assert.Equal(t, testCase.expectedAction, decideExistingWalClaim(testCase.claim)) },
		)
	}
}
