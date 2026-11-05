package streamstatistics_test

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

func (segment fixtureSegmentNo) GetFilename(timeline uint32, size uint64) string {
	segmentsPerLog := uint64(1<<32) / size
	return fmt.Sprintf("%08X%08X%08X", timeline, uint64(segment)/segmentsPerLog, uint64(segment)%segmentsPerLog)
}
