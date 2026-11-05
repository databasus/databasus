package catalog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/app/apptest"
)

func getBackups(
	t *testing.T,
	catalog *catalogFixture,
	query string,
) backupPage {
	t.Helper()

	var response backupPage
	catalog.call(
		t,
		catalogCall{
			Request:       apptest.PhysicalListRequest(catalog.DatabaseID, ""+query),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    http.StatusOK,
			Destination:   &response,
		},
	)

	return response
}

func seedOneOfEachType(t *testing.T, catalog *catalogFixture) (
	*apptest.PhysicalFullBackupSeed,
	*apptest.PhysicalIncrementalBackupSeed,
	*apptest.PhysicalWalSegmentSeed,
) {
	t.Helper()

	full := catalog.insertFull(
		t,
		newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		),
	)
	incr := catalog.insertIncremental(
		t,
		newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          full.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  fixtureLSN(segmentBytes),
				StopLSN:                   fixtureLSN(2 * segmentBytes),
			},
		),
	)
	walSegment := catalog.insertWal(
		t,
		newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000002",
				StartLSN:    fixtureLSN(2 * segmentBytes),
				EndLSN:      fixtureLSN(3 * segmentBytes),
			},
		),
	)

	return full, incr, walSegment
}

func collectBackupIDs(pages ...backupPage) []uuid.UUID {
	var backupIDs []uuid.UUID

	for _, page := range pages {
		for _, backup := range page.Backups {
			backupIDs = append(backupIDs, backup.ID)
		}
	}

	return backupIDs
}

func seedWalAroundFull(t *testing.T, catalog *catalogFixture) (
	[]*apptest.PhysicalWalSegmentSeed,
	*apptest.PhysicalFullBackupSeed,
	[]*apptest.PhysicalWalSegmentSeed,
) {
	t.Helper()

	base := time.Now().UTC().Add(-1 * time.Hour)
	olderSegments := catalog.insertContiguousWal(
		t,
		contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 2, FirstReceivedAt: base},
	)

	fullModel := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: 1,
			StartLSN:   fixtureSegmentNo(3).FirstLSN(segmentBytes) + 0x28,
			StopLSN: fixtureSegmentNo(3).
				FirstLSN(segmentBytes) +
				0x28 + 0x100,
		},
	)
	fullModel.CreatedAt = base.Add(5 * time.Minute)
	full := catalog.insertFull(t, fullModel)

	newerSegments := catalog.insertContiguousWal(
		t,
		contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 3, Count: 2, FirstReceivedAt: base.Add(10 * time.Minute)},
	)

	return olderSegments, full, newerSegments
}

func seedWalSegmentReceivedAt(
	t *testing.T,
	catalog *catalogFixture,
	segmentNo uint64,
	receivedAt time.Time,
) *apptest.PhysicalWalSegmentSeed {
	t.Helper()
	return catalog.insertContiguousWal(t, contiguousWalSpec{TimelineID: 1, FirstSegmentNo: segmentNo, Count: 1, FirstReceivedAt: receivedAt})[0]
}
