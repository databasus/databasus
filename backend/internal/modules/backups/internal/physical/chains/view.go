package chains

import (
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// ChainView is one backup chain — its FULL and its INCRs, from one FULL to the
// next. It carries no WAL: callers that need the chain's WAL look it up by Span.
type ChainView struct {
	RootFull     *catalog.Backup
	Incrementals []*catalog.Backup

	// Span is the LSN range owned by this chain along Line: from RootFull.StartLSN up to the start
	// of the next completed FULL on the line (or catalog.LSNMax if none exists yet). Only FULLs
	// bound a chain — INCRs sit inside Span, not at its edge.
	Span catalog.LSNRange
	Line TimelineLine
}

// GetOwnedSpans returns the part of each timeline the chain owns: its line cut to its span.
func (view *ChainView) GetOwnedSpans() []catalog.TimelineSpan {
	return view.Line.getSpans(view.Span)
}

// WalOrphanRef points at a WAL segment the cleaner orphan pass should sweep.
type WalOrphanRef struct {
	WalSegment *catalog.WalSegment
}
