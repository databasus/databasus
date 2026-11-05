package physical

import (
	"context"
	"time"
)

type progressSpec struct {
	Counter   *byteCounter
	Backup    BaseBackupSpec
	StartedAt time.Time
}

func reportProgress(ctx context.Context, spec progressSpec) {
	ticker := time.NewTicker(spec.Backup.ProgressInterval)
	defer ticker.Stop()
	lastReported := int64(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := spec.Counter.GetBytesWritten()
			if current == lastReported {
				continue
			}
			lastReported = current
			spec.Backup.ProgressListener(
				float64(current)/(1024*1024),
				spec.Backup.Clock.Now().Sub(spec.StartedAt).Milliseconds(),
			)
		}
	}
}
