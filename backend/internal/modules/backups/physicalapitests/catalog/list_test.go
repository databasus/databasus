package catalog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestGetPhysicalBackups(t *testing.T) {
	t.Parallel()

	t.Run("returns rows in creation order when the owner lists backups", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)

		fullModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		fullModel.CreatedAt = base
		full := catalog.insertFull(t, fullModel)

		incrModel := newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          full.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  fixtureLSN(segmentBytes),
				StopLSN:                   fixtureLSN(2 * segmentBytes),
			},
		)
		incrModel.CreatedAt = base.Add(10 * time.Minute)
		incr := catalog.insertIncremental(t, incrModel)

		walModel := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000002",
				StartLSN:    fixtureLSN(2 * segmentBytes),
				EndLSN:      fixtureLSN(3 * segmentBytes),
			},
		)
		walModel.ReceivedAt = base.Add(20 * time.Minute)
		walSegment := catalog.insertWal(t, walModel)

		var response backupPage
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalListRequest(catalog.DatabaseID, ""),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusOK,
				Destination:   &response,
			},
		)

		require.Len(t, response.Backups, 3)

		assert.Equal(t, walSegment.ID, response.Backups[0].ID)
		assert.Equal(t, "WAL", response.Backups[0].Type)
		assert.Equal(t, incr.ID, response.Backups[1].ID)
		assert.Equal(t, "INCREMENTAL", response.Backups[1].Type)
		assert.Equal(t, full.ID, response.Backups[2].ID)
		assert.Equal(t, "FULL", response.Backups[2].Type)

		assert.Equal(t, full.ID, *response.Backups[1].RootFullBackupID)
		require.NotNil(t, response.Backups[0].FromWalFilename)
		assert.Equal(t, "000000010000000000000002", *response.Backups[0].FromWalFilename)
		assert.Positive(t, response.TotalUsageMb)
	})
	t.Run("returns a failure message when a backup failed", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)

		completedModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		completedModel.CreatedAt = base
		completed := catalog.insertFull(t, completedModel)

		failReason := "PG_BASEBACKUP_FAILED"
		failedModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		failedModel.CreatedAt = base.Add(10 * time.Minute)
		failedModel.Status = "ERROR"
		failedModel.ErrorReason = &failReason
		failedModel.FailMessage = new("pg_basebackup failed; stderr: connection refused")
		failed := catalog.insertFull(t, failedModel)

		var response backupPage
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalListRequest(catalog.DatabaseID, ""),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusOK,
				Destination:   &response,
			},
		)

		require.Len(t, response.Backups, 2)

		assert.Equal(t, failed.ID, response.Backups[0].ID)
		require.NotNil(t, response.Backups[0].FailMessage)
		assert.Equal(t, "pg_basebackup failed; stderr: connection refused", *response.Backups[0].FailMessage)

		assert.Equal(t, completed.ID, response.Backups[1].ID)
		assert.Nil(t, response.Backups[1].FailMessage,
			"a completed backup carries no failure message")
	})
	t.Run("returns every row once when the caller follows cursors", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		fulls := catalog.insertFulls(t, fullsSeedSpec{Count: 5, FirstCreatedAt: time.Now().UTC().Add(-1 * time.Hour)})

		firstPage := getBackups(t, catalog, "?limit=2")
		require.Len(t, firstPage.Backups, 2)
		require.NotEmpty(t, firstPage.NextCursor)

		secondPage := getBackups(t, catalog, "?limit=2&after="+firstPage.NextCursor)
		require.Len(t, secondPage.Backups, 2)
		require.NotEmpty(t, secondPage.NextCursor)

		lastPage := getBackups(t, catalog, "?limit=2&after="+secondPage.NextCursor)
		require.Len(t, lastPage.Backups, 1)
		assert.Empty(t, lastPage.NextCursor, "the page holding the oldest row carries no cursor")

		listedIDs := collectBackupIDs(firstPage, secondPage, lastPage)
		assert.Equal(t, []uuid.UUID{fulls[4].ID, fulls[3].ID, fulls[2].ID, fulls[1].ID, fulls[0].ID}, listedIDs)
	})
	t.Run("keeps pagination stable when a backup is created between pages", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		fulls := catalog.insertFulls(t, fullsSeedSpec{Count: 3, FirstCreatedAt: time.Now().UTC().Add(-1 * time.Hour)})

		firstPage := getBackups(t, catalog, "?limit=2")
		require.NotEmpty(t, firstPage.NextCursor)

		catalog.insertFull(
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

		secondPage := getBackups(t, catalog, "?limit=2&after="+firstPage.NextCursor)

		listedIDs := collectBackupIDs(firstPage, secondPage)
		assert.Equal(t, []uuid.UUID{fulls[2].ID, fulls[1].ID, fulls[0].ID}, listedIDs)
	})
	t.Run("returns 400 when the cursor is malformed", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		for _, malformedCursor := range []string{"not-a-cursor", "e30", "eyJ0IjoieCJ9"} {
			catalog.call(
				t,
				catalogCall{
					Request:       apptest.PhysicalListRequest(catalog.DatabaseID, "?after="+malformedCursor),
					Authorization: "Bearer " + catalog.Owner.Token,
					StatusCode:    http.StatusBadRequest,
				},
			)
		}
	})
	t.Run("lists only the requested database when a cursor comes from another database", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)
		ownFulls := catalog.insertFulls(t, fullsSeedSpec{Count: 2, FirstCreatedAt: base})

		other := createSecondPhysicalDatabase(t, catalog)
		for index := range 2 {
			otherModel := newFullSeed(
				fullSeedSpec{
					DatabaseID: other.ID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   fixtureLSN(0),
					StopLSN:    fixtureLSN(segmentBytes),
				},
			)
			otherModel.CreatedAt = base.Add(time.Duration(30+index) * time.Minute)
			catalog.insertFull(t, otherModel)
		}

		var otherPage backupPage
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalListRequest(other.ID, "?limit=1"),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusOK,
				Destination:   &otherPage,
			},
		)
		require.NotEmpty(t, otherPage.NextCursor)

		page := getBackups(t, catalog, "?after="+otherPage.NextCursor)

		assert.Equal(t, []uuid.UUID{ownFulls[1].ID, ownFulls[0].ID}, collectBackupIDs(page))
	})
	t.Run("returns 404 when a nonmember uses a cursor", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		catalog.insertFulls(t, fullsSeedSpec{Count: 2, FirstCreatedAt: time.Now().UTC().Add(-1 * time.Hour)})

		firstPage := getBackups(t, catalog, "?limit=1")
		require.NotEmpty(t, firstPage.NextCursor)

		outsider := catalog.App.CreateNonAdminUser(t, catalog.Admin)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalListRequest(catalog.DatabaseID, "?after="+firstPage.NextCursor),
				Authorization: "Bearer " + outsider.Token,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("returns 404 when the user is not a workspace member", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		outsider := catalog.App.CreateNonAdminUser(t, catalog.Admin)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalListRequest(catalog.DatabaseID, ""),
				Authorization: "Bearer " + outsider.Token,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("filters by type when one type is requested", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full, incr, walSegment := seedOneOfEachType(t, catalog)

		fullPage := getBackups(t, catalog, "?type=FULL")
		require.Len(t, fullPage.Backups, 1)
		assert.Equal(t, full.ID, fullPage.Backups[0].ID)

		incrPage := getBackups(t, catalog, "?type=INCREMENTAL")
		require.Len(t, incrPage.Backups, 1)
		assert.Equal(t, incr.ID, incrPage.Backups[0].ID)

		walPage := getBackups(t, catalog, "?type=WAL")
		require.Len(t, walPage.Backups, 1)
		assert.Equal(t, walSegment.ID, walPage.Backups[0].ID)
	})
	t.Run("filters by status when one status is requested", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		completed := catalog.insertFull(
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
		inProgress := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		completedPage := getBackups(t, catalog, "?status=COMPLETED")
		require.Len(t, completedPage.Backups, 1)
		assert.Equal(t, completed.ID, completedPage.Backups[0].ID)

		inProgressPage := getBackups(t, catalog, "?status=IN_PROGRESS")
		require.Len(t, inProgressPage.Backups, 1)
		assert.Equal(t, inProgress.ID, inProgressPage.Backups[0].ID)
	})
	t.Run("excludes newer rows when a date cutoff is supplied", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		now := time.Now().UTC()

		olderModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		olderModel.CreatedAt = now.Add(-3 * time.Hour)
		older := catalog.insertFull(t, olderModel)

		newerModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(2 * segmentBytes),
				StopLSN:    fixtureLSN(3 * segmentBytes),
			},
		)
		newerModel.CreatedAt = now
		catalog.insertFull(t, newerModel)

		cutoff := now.Add(-1 * time.Hour).Format(time.RFC3339)
		page := getBackups(t, catalog, "?createdBefore="+cutoff)

		require.Len(t, page.Backups, 1)
		assert.Equal(t, older.ID, page.Backups[0].ID)
	})
	t.Run("accepts several types when the filter repeats", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full, _, walSegment := seedOneOfEachType(t, catalog)

		page := getBackups(t, catalog, "?type=FULL&type=WAL")

		require.Len(t, page.Backups, 2)

		returnedIDs := []uuid.UUID{page.Backups[0].ID, page.Backups[1].ID}
		assert.Contains(t, returnedIDs, full.ID)
		assert.Contains(t, returnedIDs, walSegment.ID)
	})
	t.Run("accepts several statuses when the filter repeats", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		completed := catalog.insertFull(
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
		inProgress := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		erroredModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(2 * segmentBytes),
				StopLSN:    fixtureLSN(3 * segmentBytes),
			},
		)
		erroredModel.Status = "ERROR"
		catalog.insertFull(t, erroredModel)

		page := getBackups(t, catalog, "?status=COMPLETED&status=IN_PROGRESS")

		require.Len(t, page.Backups, 2)

		returnedIDs := []uuid.UUID{page.Backups[0].ID, page.Backups[1].ID}
		assert.Contains(t, returnedIDs, completed.ID)
		assert.Contains(t, returnedIDs, inProgress.ID)
	})
	t.Run("combines type and status filters when both are supplied", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		completedFull := catalog.insertFull(
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

		catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		catalog.insertIncremental(
			t,
			newIncrementalSeed(
				incrementalSeedSpec{
					DatabaseID:                catalog.DatabaseID,
					StorageID:                 catalog.StorageID,
					RootFullBackupID:          completedFull.ID,
					ParentIncrementalBackupID: nil,
					TimelineID:                1,
					StartLSN:                  fixtureLSN(segmentBytes),
					StopLSN:                   fixtureLSN(2 * segmentBytes),
				},
			),
		)

		page := getBackups(t, catalog, "?type=FULL&status=COMPLETED")

		require.Len(t, page.Backups, 1)
		assert.Equal(t, completedFull.ID, page.Backups[0].ID)
	})
	t.Run("returns an empty page when the database has no backups", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		page := getBackups(t, catalog, "")

		assert.Empty(t, page.Backups)
		assert.Empty(t, page.NextCursor)
		assert.Zero(t, page.TotalUsageMb)
	})
	t.Run("isolates rows when another database has backups", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		ownFull := catalog.insertFull(
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

		other := createSecondPhysicalDatabase(t, catalog)
		catalog.insertFull(
			t,
			newFullSeed(
				fullSeedSpec{
					DatabaseID: other.ID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   fixtureLSN(0),
					StopLSN:    fixtureLSN(segmentBytes),
				},
			),
		)

		page := getBackups(t, catalog, "")

		require.Len(t, page.Backups, 1)
		assert.Equal(t, ownFull.ID, page.Backups[0].ID)
	})
	t.Run("uses the default or maximum page size when the caller requests it", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		createCompletedFullBackupsInBulk(t, catalog, 1001)

		unset := getBackups(t, catalog, "")
		assert.Len(t, unset.Backups, 50)
		assert.NotEmpty(t, unset.NextCursor)

		maximumPage := getBackups(t, catalog, "?limit=1000")
		assert.Len(t, maximumPage.Backups, 1000)
		assert.NotEmpty(t, maximumPage.NextCursor)
	})
	t.Run("groups consecutive WAL segments when their timeline matches", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)

		fullModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		fullModel.CreatedAt = base
		full := catalog.insertFull(t, fullModel)

		segments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 5, FirstReceivedAt: base.Add(time.Minute)},
		)

		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 2)

		walRange := page.Backups[0]
		assert.Equal(t, "WAL", walRange.Type)
		assert.Equal(t, segments[0].ID, walRange.ID)
		assert.Equal(t, segments[0].WalFilename, *walRange.FromWalFilename)
		assert.Equal(t, segments[4].WalFilename, *walRange.ToWalFilename)
		assert.Equal(t, 5, *walRange.WalSegmentsCount)
		assert.InDelta(t, 80, walRange.SizeMb, 0.001)
		assert.Equal(t, segments[0].StartLSN, walRange.StartLSN)
		assert.Equal(t, segments[4].EndLSN, walRange.StopLSN)
		assert.WithinDuration(t, segments[0].ReceivedAt, walRange.CreatedAt, time.Millisecond)
		require.NotNil(t, walRange.CompletedAt)
		assert.WithinDuration(t, segments[4].ReceivedAt, *walRange.CompletedAt, time.Millisecond)
		assert.False(t, walRange.HasGapBefore)

		assert.Equal(t, full.ID, page.Backups[1].ID)
	})
	t.Run("splits WAL ranges and marks the gap when a segment is missing", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)
		olderSegments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 2, FirstReceivedAt: base},
		)
		newerSegments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 4, Count: 2, FirstReceivedAt: base.Add(10 * time.Minute)},
		)

		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 2)

		assert.Equal(t, newerSegments[0].ID, page.Backups[0].ID)
		assert.True(t, page.Backups[0].HasGapBefore, "the range after the missing segment is marked")

		assert.Equal(t, olderSegments[0].ID, page.Backups[1].ID)
		assert.False(t, page.Backups[1].HasGapBefore, "the oldest retained range has nothing before it")
	})
	t.Run("starts a range without a gap mark when the timeline changes", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)
		oldTimelineSegments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 2, FirstReceivedAt: base},
		)
		newTimelineSegments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 2, FirstSegmentNo: 3, Count: 2, FirstReceivedAt: base.Add(10 * time.Minute)},
		)

		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 2)

		assert.Equal(t, newTimelineSegments[0].ID, page.Backups[0].ID)
		assert.Equal(t, 2, page.Backups[0].TimelineID)
		assert.False(t, page.Backups[0].HasGapBefore, "the first range on a timeline is not a gap")

		assert.Equal(t, oldTimelineSegments[0].ID, page.Backups[1].ID)
		assert.Equal(t, 1, page.Backups[1].TimelineID)
	})
	t.Run("splits a WAL range when a full backup lies between segments", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		olderSegments, full, newerSegments := seedWalAroundFull(t, catalog)

		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 3)

		assert.Equal(t, newerSegments[0].ID, page.Backups[0].ID)
		assert.Equal(t, 2, *page.Backups[0].WalSegmentsCount)
		assert.False(t, page.Backups[0].HasGapBefore)

		assert.Equal(t, full.ID, page.Backups[1].ID)

		assert.Equal(t, olderSegments[0].ID, page.Backups[2].ID)
		assert.Equal(t, 2, *page.Backups[2].WalSegmentsCount)
	})
	t.Run("merges WAL ranges across a full when the caller filters to WAL", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		olderSegments, _, newerSegments := seedWalAroundFull(t, catalog)

		page := getBackups(t, catalog, "?type=WAL")
		require.Len(t, page.Backups, 1)

		assert.Equal(t, olderSegments[0].ID, page.Backups[0].ID)
		assert.Equal(t, 4, *page.Backups[0].WalSegmentsCount)
		assert.Equal(t, newerSegments[1].WalFilename, *page.Backups[0].ToWalFilename)
	})
	t.Run("limits a WAL range when the date cutoff falls inside it", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)
		segments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 4, FirstReceivedAt: base},
		)

		cutoff := segments[1].ReceivedAt.Add(30 * time.Second).Format(time.RFC3339Nano)
		page := getBackups(t, catalog, "?createdBefore="+cutoff)
		require.Len(t, page.Backups, 1)

		assert.Equal(t, 2, *page.Backups[0].WalSegmentsCount)
		assert.Equal(t, segments[1].WalFilename, *page.Backups[0].ToWalFilename)
		assert.InDelta(t, 32, page.Backups[0].SizeMb, 0.001)
	})
	t.Run("lists no WAL when the requested status cannot describe WAL", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)
		catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 2, FirstReceivedAt: base},
		)

		failedModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		failedModel.Status = "ERROR"
		failed := catalog.insertFull(t, failedModel)

		page := getBackups(t, catalog, "?status=ERROR")

		assert.Equal(t, []uuid.UUID{failed.ID}, collectBackupIDs(page))
	})
	t.Run("grows a WAL range in place when a segment is appended", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-1 * time.Hour)
		segments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 2, FirstReceivedAt: base},
		)

		pageBefore := getBackups(t, catalog, "")
		require.Len(t, pageBefore.Backups, 1)

		appendedSegment := catalog.insertContiguousWal(t, contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 3, Count: 1, FirstReceivedAt: base.Add(5 * time.Minute)})[0]

		pageAfter := getBackups(t, catalog, "")
		require.Len(t, pageAfter.Backups, 1)

		grownRange := pageAfter.Backups[0]
		assert.Equal(t, segments[0].ID, grownRange.ID)
		assert.Equal(t, 3, *grownRange.WalSegmentsCount)
		assert.Equal(t, appendedSegment.WalFilename, *grownRange.ToWalFilename)
		assert.Greater(t, grownRange.SizeMb, pageBefore.Backups[0].SizeMb)
		assert.WithinDuration(t, appendedSegment.ReceivedAt, *grownRange.CompletedAt, time.Millisecond)
	})
	t.Run("counts a WAL range as one row when it contains many segments", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-2 * time.Hour)

		fullModel := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		fullModel.CreatedAt = base
		full := catalog.insertFull(t, fullModel)

		segments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{TimelineID: 1, FirstSegmentNo: 1, Count: 60, FirstReceivedAt: base.Add(time.Minute)},
		)

		page := getBackups(t, catalog, "?limit=2")

		assert.Equal(t, []uuid.UUID{segments[0].ID, full.ID}, collectBackupIDs(page))
		assert.Equal(t, 60, *page.Backups[0].WalSegmentsCount)
		assert.Empty(t, page.NextCursor)
	})
}
