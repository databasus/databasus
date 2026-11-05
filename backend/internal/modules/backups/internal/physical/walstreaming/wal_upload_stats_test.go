package walstreaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestArchivingStats(t *testing.T) {
	t.Run("reports interval totals and resets them when a snapshot is taken", func(t *testing.T) {
		statistics := archivingStats{}
		statistics.recordArchived(
			100,
			segmentUploadTimings{
				Claim:        time.Millisecond,
				SegmentWrite: 3 * time.Millisecond,
				SidecarWrite: 2 * time.Millisecond,
				Commit:       time.Millisecond,
			},
		)
		statistics.recordArchived(
			200,
			segmentUploadTimings{
				Claim:        3 * time.Millisecond,
				SegmentWrite: 5 * time.Millisecond,
				SidecarWrite: 4 * time.Millisecond,
				Commit:       3 * time.Millisecond,
			},
		)
		statistics.recordConcurrentUploads(3)
		statistics.recordConcurrentUploads(2)

		snapshot := statistics.takeSnapshot()

		assert.Equal(t, int64(2), snapshot.ArchivedSegments)
		assert.Equal(t, int64(300), snapshot.ArchivedBytes)
		assert.Equal(t, 3, snapshot.HighestConcurrentUploads)
		assert.Equal(t, 4*time.Millisecond, snapshot.Claim.Total)
		assert.Equal(t, 3*time.Millisecond, snapshot.Claim.Longest)
		assert.Equal(t, 2*time.Millisecond, snapshot.Claim.GetAverage(snapshot.ArchivedSegments))
		assert.Equal(t, 8*time.Millisecond, snapshot.SegmentWrite.Total)
		assert.Equal(t, 5*time.Millisecond, snapshot.SegmentWrite.Longest)
		assert.Equal(t, 6*time.Millisecond, snapshot.SidecarWrite.Total)
		assert.Equal(t, 4*time.Millisecond, snapshot.Commit.Total)
		assert.Equal(t, ArchivingSnapshot{}, statistics.takeSnapshot())
	})
}
