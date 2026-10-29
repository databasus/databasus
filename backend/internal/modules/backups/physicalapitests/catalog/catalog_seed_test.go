package catalog_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/app/apptest"
)

const segmentBytes = 16 * 1024 * 1024

type fixtureLSN uint64

func (lsn fixtureLSN) String() string {
	return fmt.Sprintf("%X/%X", uint64(lsn)>>32, uint64(lsn)&0xffffffff)
}

type fixtureSegmentNo uint64

func (segment fixtureSegmentNo) FirstLSN(size uint64) fixtureLSN {
	return fixtureLSN(uint64(segment) * size)
}

type fullSeedSpec struct {
	DatabaseID uuid.UUID
	StorageID  uuid.UUID
	TimelineID int
	StartLSN   fixtureLSN
	StopLSN    fixtureLSN
}

func newFullSeed(spec fullSeedSpec) *apptest.PhysicalFullBackupSeed {
	return &apptest.PhysicalFullBackupSeed{
		DatabaseID: spec.DatabaseID, StorageID: spec.StorageID,
		TimelineID: spec.TimelineID, Status: "COMPLETED", FileName: new(uuid.NewString() + ".zst"),
		StartLSN: new(spec.StartLSN.String()), StopLSN: new(spec.StopLSN.String()), CreatedAt: time.Now().UTC(),
	}
}

type inProgressFullSpec struct {
	DatabaseID uuid.UUID
	StorageID  uuid.UUID
	TimelineID int
}

func newInProgressFull(spec inProgressFullSpec) *apptest.PhysicalFullBackupSeed {
	return &apptest.PhysicalFullBackupSeed{
		DatabaseID: spec.DatabaseID, StorageID: spec.StorageID,
		TimelineID: spec.TimelineID, Status: "IN_PROGRESS", CreatedAt: time.Now().UTC(),
	}
}

type incrementalSeedSpec struct {
	DatabaseID                uuid.UUID
	StorageID                 uuid.UUID
	RootFullBackupID          uuid.UUID
	ParentIncrementalBackupID *uuid.UUID
	TimelineID                int
	StartLSN                  fixtureLSN
	StopLSN                   fixtureLSN
}

func newIncrementalSeed(spec incrementalSeedSpec) *apptest.PhysicalIncrementalBackupSeed {
	return &apptest.PhysicalIncrementalBackupSeed{
		DatabaseID: spec.DatabaseID, StorageID: spec.StorageID,
		RootFullBackupID: spec.RootFullBackupID, ParentIncrementalBackupID: spec.ParentIncrementalBackupID,
		TimelineID: spec.TimelineID, Status: "COMPLETED", FileName: new(uuid.NewString() + ".zst"),
		StartLSN: new(spec.StartLSN.String()), StopLSN: new(spec.StopLSN.String()), CreatedAt: time.Now().UTC(),
	}
}

type walSeedSpec struct {
	DatabaseID  uuid.UUID
	StorageID   uuid.UUID
	TimelineID  int
	WalFilename string
	StartLSN    fixtureLSN
	EndLSN      fixtureLSN
}

func newWalSeed(spec walSeedSpec) *apptest.PhysicalWalSegmentSeed {
	return &apptest.PhysicalWalSegmentSeed{
		DatabaseID:       spec.DatabaseID,
		StorageID:        spec.StorageID,
		TimelineID:       spec.TimelineID,
		WalFilename:      spec.WalFilename,
		FileName:         new(uuid.NewString() + ".zst"),
		StartLSN:         spec.StartLSN.String(),
		EndLSN:           spec.EndLSN.String(),
		CompressedSizeMb: 16,
		ReceivedAt:       time.Now().UTC(),
	}
}

func (catalog *catalogFixture) insertFull(
	t *testing.T,
	seed *apptest.PhysicalFullBackupSeed,
) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	*seed = catalog.App.InsertPhysicalFullBackup(t, *seed)
	return seed
}

func (catalog *catalogFixture) insertIncremental(
	t *testing.T,
	seed *apptest.PhysicalIncrementalBackupSeed,
) *apptest.PhysicalIncrementalBackupSeed {
	t.Helper()
	*seed = catalog.App.InsertPhysicalIncrementalBackup(t, *seed)
	return seed
}

func (catalog *catalogFixture) insertWal(
	t *testing.T,
	seed *apptest.PhysicalWalSegmentSeed,
) *apptest.PhysicalWalSegmentSeed {
	t.Helper()
	*seed = catalog.App.InsertPhysicalWalSegment(t, *seed)
	return seed
}

type fullsSeedSpec struct {
	Count          int
	FirstCreatedAt time.Time
}

func (catalog *catalogFixture) insertFulls(t *testing.T, spec fullsSeedSpec) []*apptest.PhysicalFullBackupSeed {
	t.Helper()
	fulls := make([]*apptest.PhysicalFullBackupSeed, 0, spec.Count)
	for index := range spec.Count {
		seed := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		seed.CreatedAt = spec.FirstCreatedAt.Add(time.Duration(index) * time.Minute)
		fulls = append(fulls, catalog.insertFull(t, seed))
	}
	return fulls
}

type contiguousWalSpec struct {
	TimelineID      int
	FirstSegmentNo  uint64
	Count           int
	FirstReceivedAt time.Time
}

func (catalog *catalogFixture) insertContiguousWal(
	t *testing.T,
	spec contiguousWalSpec,
) []*apptest.PhysicalWalSegmentSeed {
	t.Helper()
	segments := make([]*apptest.PhysicalWalSegmentSeed, 0, spec.Count)
	for index := range spec.Count {
		segmentNo := spec.FirstSegmentNo + uint64(index)
		seed := newWalSeed(walSeedSpec{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: spec.TimelineID,
			WalFilename: fmt.Sprintf("%08X%08X%08X", spec.TimelineID, segmentNo/256, segmentNo%256),
			StartLSN:    fixtureLSN(segmentNo * segmentBytes), EndLSN: fixtureLSN((segmentNo + 1) * segmentBytes),
		})
		seed.ReceivedAt = spec.FirstReceivedAt.Add(time.Duration(index) * time.Minute)
		segments = append(segments, catalog.insertWal(t, seed))
	}
	return segments
}

func createCompletedFullBackupsInBulk(t *testing.T, catalog *catalogFixture, count int) {
	t.Helper()
	catalog.App.ExecuteSQL(t, `INSERT INTO backups_physical
 (database_id,storage_id,timeline_id,status,start_lsn,stop_lsn,created_at,completed_at,kind,file_name)
 SELECT ?,?,1,'COMPLETED','0/0'::pg_lsn,'0/100'::pg_lsn,
 NOW()-make_interval(secs=>series_index),NOW()-make_interval(secs=>series_index),'FULL','bulk-'||series_index
 FROM generate_series(1,?) AS series_index`, catalog.DatabaseID, catalog.StorageID, count)
}
