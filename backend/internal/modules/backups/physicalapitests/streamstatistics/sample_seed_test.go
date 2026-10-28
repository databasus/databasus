package streamstatistics_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/app/apptest"
)

const sampleInterval = 30 * time.Second

type samplesSeedSpec struct {
	DatabaseID uuid.UUID
	Count      int
	NewestAt   time.Time
	Modify     func(*apptest.PhysicalWalStreamSampleSeed)
}

func newStatisticsCatalog(t *testing.T) *catalogFixture {
	t.Helper()
	catalog := newCatalog(t)
	apptest.SaveBackupConfigWithStorage(
		t,
		catalog.Owner,
		apptest.StorageUse{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID},
	)
	return catalog
}

func newStreamSample(databaseID uuid.UUID, sampledAt time.Time) *apptest.PhysicalWalStreamSampleSeed {
	return &apptest.PhysicalWalStreamSampleSeed{
		DatabaseID:               databaseID,
		SampledAt:                sampledAt,
		SessionStartedAt:         sampledAt.Add(-time.Hour),
		SourceState:              "REACHABLE",
		SourceCurrentLSN:         new("0/50000000"),
		ArchiveProgressLSN:       new("0/4F000000"),
		SourceWrittenBytes:       new(int64(segmentBytes)),
		ArchivedSegments:         1,
		ArchivedWalBytes:         segmentBytes,
		ArchivedStoredBytes:      4 * 1024 * 1024,
		WalSegmentSizeBytes:      segmentBytes,
		HighestConcurrentUploads: 1,
		SegmentWriteAvgMillis:    new(int64(100)),
		SegmentWriteMaxMillis:    new(int64(150)),
		SlotWalStatus: new(
			"reserved",
		),
		SlotLagBytes:         new(int64(1024 * 1024)),
		SlotSafeWalSizeBytes: new(int64(10 * 1024 * 1024 * 1024)),
		UnarchivedWalBytes: new(
			int64(segmentBytes),
		),
		LocalQueueBytes:      segmentBytes,
		LocalQueuePauseBytes: 512 * 1024 * 1024,
		IsReceiverRunning:    true,
	}
}

func (catalog *catalogFixture) insertSample(
	t *testing.T,
	sample *apptest.PhysicalWalStreamSampleSeed,
) *apptest.PhysicalWalStreamSampleSeed {
	t.Helper()
	*sample = catalog.App.InsertPhysicalWalStreamSample(t, *sample)
	return sample
}

func (catalog *catalogFixture) insertSamples(
	t *testing.T,
	spec samplesSeedSpec,
) []*apptest.PhysicalWalStreamSampleSeed {
	t.Helper()
	samples := make([]*apptest.PhysicalWalStreamSampleSeed, 0, spec.Count)
	for sampleIndex := range spec.Count {
		sample := newStreamSample(
			spec.DatabaseID,
			spec.NewestAt.Add(-time.Duration(spec.Count-1-sampleIndex)*sampleInterval),
		)
		if spec.Modify != nil {
			spec.Modify(sample)
		}
		samples = append(samples, catalog.insertSample(t, sample))
	}
	return samples
}

func (catalog *catalogFixture) insertStreamer(
	t *testing.T,
	streamer *apptest.PhysicalWalStreamerSeed,
) *apptest.PhysicalWalStreamerSeed {
	t.Helper()
	*streamer = catalog.App.InsertPhysicalWalStreamer(t, *streamer)
	return streamer
}
