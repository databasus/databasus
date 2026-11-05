package physical

import (
	"context"
	"time"
)

type byteStallSpec struct {
	Counter   *byteCounter
	Backup    BaseBackupSpec
	StartedAt time.Time
	OnStall   func()
}

func watchByteStall(ctx context.Context, spec byteStallSpec) {
	ticker := time.NewTicker(spec.Backup.StallPollInterval)
	defer ticker.Stop()
	lastBytes := int64(0)
	lastProgressAt := spec.StartedAt
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := spec.Backup.Clock.Now()
			current := spec.Counter.GetBytesWritten()
			if current != lastBytes {
				lastBytes = current
				lastProgressAt = now
				continue
			}

			budget := spec.Backup.StallTimeout
			if current == 0 {
				budget = spec.Backup.FirstByteTimeout
			}
			if now.Sub(lastProgressAt) > budget {
				spec.OnStall()
				return
			}
		}
	}
}
