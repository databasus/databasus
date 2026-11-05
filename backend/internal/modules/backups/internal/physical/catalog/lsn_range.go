package catalog

import "databasus-backend/internal/engines/postgresql/physical/walmath"

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
