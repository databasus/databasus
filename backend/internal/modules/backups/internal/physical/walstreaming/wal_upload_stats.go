package walstreaming

import (
	"sync"
	"time"
)

type uploadStepTiming struct {
	Total   time.Duration
	Longest time.Duration
}

func (t uploadStepTiming) GetAverage(count int64) time.Duration {
	if count == 0 {
		return 0
	}

	return t.Total / time.Duration(count)
}

func (t *uploadStepTiming) record(duration time.Duration) {
	t.Total += duration
	t.Longest = max(t.Longest, duration)
}

type segmentUploadTimings struct {
	Claim        time.Duration
	SegmentWrite time.Duration
	SidecarWrite time.Duration
	Commit       time.Duration
}

type ArchivingSnapshot struct {
	ArchivedSegments int64
	ArchivedBytes    int64

	Claim        uploadStepTiming
	SegmentWrite uploadStepTiming
	SidecarWrite uploadStepTiming
	Commit       uploadStepTiming

	HighestConcurrentUploads int
}

type archivingStats struct {
	mutex    sync.Mutex
	snapshot ArchivingSnapshot
}

func (s *archivingStats) recordArchived(compressedSizeBytes int64, timings segmentUploadTimings) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.snapshot.ArchivedSegments++
	s.snapshot.ArchivedBytes += compressedSizeBytes
	s.snapshot.Claim.record(timings.Claim)
	s.snapshot.SegmentWrite.record(timings.SegmentWrite)
	s.snapshot.SidecarWrite.record(timings.SidecarWrite)
	s.snapshot.Commit.record(timings.Commit)
}

func (s *archivingStats) recordConcurrentUploads(count int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.snapshot.HighestConcurrentUploads = max(s.snapshot.HighestConcurrentUploads, count)
}

func (s *archivingStats) takeSnapshot() ArchivingSnapshot {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	taken := s.snapshot
	s.snapshot = ArchivingSnapshot{}

	return taken
}
