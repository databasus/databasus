package chain_view

import (
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

// ChainView is one backup chain — its FULL and its INCRs, from one FULL to the
// next. It carries no WAL: callers that need the chain's WAL look it up by Span.
type ChainView struct {
	RootFull     *physical_models.PhysicalFullBackup
	Incrementals []*physical_models.PhysicalIncrementalBackup

	// Span is the LSN range owned by this chain: from RootFull.StartLSN up to
	// the next FULL's StartLSN (or LSNMax if no next FULL exists yet). Only
	// FULLs bound a chain — INCRs sit inside Span, not at its edge.
	Span LSNRange
}

// LSNMax is the unbounded upper sentinel for the latest chain on a timeline
// (no successor FULL exists). Encoded as the maximum uint64 LSN so range
// predicates against pg_lsn columns still work without a special case.
const LSNMax walmath.LSN = ^walmath.LSN(0)

type LSNRange struct {
	Start walmath.LSN
	End   walmath.LSN
}

func (r LSNRange) Contains(lsn walmath.LSN) bool {
	return lsn >= r.Start && lsn < r.End
}

func (r LSNRange) IsUnbounded() bool {
	return r.End == LSNMax
}

type ValidationResult struct {
	Status  ValidationStatus
	Message string
}

// WalOrphanRef points at a WAL segment the cleaner orphan pass should sweep.
type WalOrphanRef struct {
	WalSegment *physical_models.PhysicalWalSegment
}
