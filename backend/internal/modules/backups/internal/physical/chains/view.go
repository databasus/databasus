package chains

import (
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// ChainView is one backup chain — its FULL and its INCRs, from one FULL to the
// next. It carries no WAL: callers that need the chain's WAL look it up by Span.
type ChainView struct {
	RootFull     *catalog.Backup
	Incrementals []*catalog.Backup

	// Span is the LSN range owned by this chain: from RootFull.StartLSN up to
	// the next FULL's StartLSN (or catalog.LSNMax if no next FULL exists yet). Only
	// FULLs bound a chain — INCRs sit inside Span, not at its edge.
	Span catalog.LSNRange
}

// WalOrphanRef points at a WAL segment the cleaner orphan pass should sweep.
type WalOrphanRef struct {
	WalSegment *catalog.WalSegment
}
