package chains

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type DeleteOrphanWalSegmentsSpec struct {
	DatabaseID      uuid.UUID
	TimelineID      int
	Span            catalog.LSNRange
	WalByteBudgetMB float64
}

// DeletedSummary reports what a single DeleteFull / DeleteChainDependentsKeepFull
// call removed. The cleaner logs it per tick. ChainFullyDeleted is false when
// the per-tick WAL byte budget capped the call before the FULL row was reached
// (or when the call intentionally keeps the FULL) — the caller resumes on the
// next tick.
type DeletedSummary struct {
	RootFullBackupID  uuid.UUID
	WalSegments       int
	Incrementals      int
	HistoryFiles      int
	BytesDeletedMB    float64
	ChainFullyDeleted bool
}
