package chains

import (
	"errors"
	"fmt"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// ErrNoChainForRestore means the database has no COMPLETED FULL backup to
// restore from at all.
var ErrNoChainForRestore = errors.New("chain_view: no completed full backup to restore from")

// ErrTargetBeforeEarliest means the requested point-in-time precedes the
// earliest FULL backup — an older chain would be needed but none exists.
var ErrTargetBeforeEarliest = errors.New("chain_view: target time precedes the earliest full backup")

// WalGapBeforeTargetError means the contiguous WAL we can actually ship does not
// reach the requested target: a missing/in-flight segment breaks the replay
// run before the target. The restore is refused rather than silently truncated
// (segments CAN exist past a gap in storage, so the resolver must reject
// explicitly instead of trusting their absence). The carried values let the
// caller tell the user the furthest point that IS restorable.
type WalGapBeforeTargetError struct {
	LatestRestorableLSN walmath.LSN
}

func (e WalGapBeforeTargetError) Error() string {
	return fmt.Sprintf(
		"chain_view: wal gap before target; latest restorable point is %s",
		e.LatestRestorableLSN.String(),
	)
}
