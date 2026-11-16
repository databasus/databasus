package chains

import (
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type ChainExtension struct {
	RootFull       *catalog.Backup
	NewestBackup   *catalog.Backup
	LiveTimelineID int
}

// CanExtendChain reports whether an INCREMENTAL taken on the live timeline can join the chain: the
// root FULL's line ends on the live timeline, and the chain's newest backup stopped on that line,
// at or before the fork point where the line leaves its timeline. Restore then replays the chain
// and the WAL across every fork point in between.
func CanExtendChain(histories []*catalog.WalHistoryFile, extension ChainExtension) bool {
	line := BuildTimelineLine(histories, extension.RootFull)
	if line.IsOffLine || line.Ranges[len(line.Ranges)-1].TimelineID != extension.LiveTimelineID {
		return false
	}

	newestBackup := extension.NewestBackup

	return line.getRangeIndexHoldingStop(newestBackup.TimelineID, *newestBackup.StopLSN) >= 0
}
