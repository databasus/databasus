package restoring_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSelectPhysicalRestoreArtifacts(t *testing.T) {
	t.Parallel()

	t.Run("ships only the selected full without WAL", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full := catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: timeline.at(0)})
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 1, ReceivedAt: timeline.at(5)})
		entries := catalog.downloadRestore(t, restoreRequest{BackupID: &full.ID})
		assertRestoreArtifacts(t, entries, full.ID, nil)
	})
	t.Run("ships the selected incremental and its ancestors without later incrementals or WAL", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full, first, second := seedFullTwoIncrementals(t, catalog, timeline)
		catalog.seedRestoreIncremental(
			t,
			restoreIncrementalSpec{FullID: full.ID, ParentID: &second.ID, Segment: 3, CompletedAt: timeline.at(30)},
		)
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 2, ReceivedAt: timeline.at(12)})
		entries := catalog.downloadRestore(t, restoreRequest{BackupID: &second.ID})
		assertRestoreArtifacts(t, entries, full.ID, []uuid.UUID{first.ID, second.ID})
	})
	t.Run("returns 404 for a missing backup", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		backupID := uuid.New()
		catalog.downloadRestore(t, restoreRequest{BackupID: &backupID, StatusCode: http.StatusNotFound})
	})
	t.Run("returns 409 when there is no full backup", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.downloadRestore(t, restoreRequest{StatusCode: http.StatusConflict})
	})
	t.Run("ships the latest full with no WAL when its replay window is empty", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full := catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: timeline.at(0)})
		entries := catalog.downloadRestore(t, restoreRequest{})
		assertRestoreArtifacts(t, entries, full.ID, nil)
	})
	t.Run("ships WAL only after the last included incremental", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full, first, second := seedFullTwoIncrementals(t, catalog, timeline)
		for segment, minute := range map[int]int{2: 12, 3: 25, 4: 30} {
			catalog.seedRestoreWal(t, restoreWalSpec{Segment: segment, ReceivedAt: timeline.at(minute)})
		}
		entries := catalog.downloadRestore(t, restoreRequest{})
		assertRestoreArtifacts(t, entries, full.ID, []uuid.UUID{first.ID, second.ID}, 3, 4)
	})
	t.Run("uses the earlier incremental and all replay WAL for a target between incrementals", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full, first, _ := seedFullTwoIncrementals(t, catalog, timeline)
		for segment, minute := range map[int]int{2: 12, 3: 25, 4: 30} {
			catalog.seedRestoreWal(t, restoreWalSpec{Segment: segment, ReceivedAt: timeline.at(minute)})
		}
		entries := catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(15))})
		assertRestoreArtifacts(t, entries, full.ID, []uuid.UUID{first.ID}, 2, 3, 4)
	})
	t.Run("uses the older chain and its WAL for a target during a newer full", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		older := catalog.seedRestoreFull(
			t,
			restoreFullSpec{StartLSN: lsnAt(0) + 0x28, StopLSN: lsnAt(1) + 0x28, CompletedAt: timeline.at(0)},
		)
		catalog.seedRestoreFull(
			t,
			restoreFullSpec{StartLSN: lsnAt(2) + 0x28, StopLSN: lsnAt(2) + 0x128, CompletedAt: timeline.at(10)},
		)
		for segment, minute := range map[int]int{1: 4, 2: 6, 3: 11, 4: 12} {
			catalog.seedRestoreWal(t, restoreWalSpec{Segment: segment, ReceivedAt: timeline.at(minute)})
		}
		entries := catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(7))})
		assertRestoreArtifacts(t, entries, older.ID, nil, 1, 2, 3, 4)
	})
	t.Run("returns 409 for a target beyond a WAL gap", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		seedFullTwoIncrementals(t, catalog, timeline)
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 2, ReceivedAt: timeline.at(12)})
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 4, ReceivedAt: timeline.at(30)})

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, new(timeline.at(25))),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusConflict,
				Destination:   &problem,
			},
		)

		assert.Equal(t, apiclient.ProblemCodeTargetTimeUnreachable, problem.Code)
		assert.Contains(t, problem.Detail, "0/3000000")
	})
	t.Run("clamps a latest restore to the contiguous WAL before the gap", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full, first, second := seedFullTwoIncrementals(t, catalog, timeline)
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 3, ReceivedAt: timeline.at(25)})
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 5, ReceivedAt: timeline.at(40)})
		entries := catalog.downloadRestore(t, restoreRequest{})
		assertRestoreArtifacts(t, entries, full.ID, []uuid.UUID{first.ID, second.ID}, 3)
	})
	t.Run("includes the WAL segment covering a full that ends within a segment", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full := catalog.seedRestoreFull(t, restoreFullSpec{StartLSN: 1000, StopLSN: 2000, CompletedAt: timeline.at(0)})
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 0, ReceivedAt: timeline.at(5)})
		catalog.seedRestoreWal(t, restoreWalSpec{Segment: 1, ReceivedAt: timeline.at(10)})
		entries := catalog.downloadRestore(t, restoreRequest{})
		assertRestoreArtifacts(t, entries, full.ID, nil, 0, 1)
	})
	t.Run("returns 409 for a target before the earliest full", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: timeline.at(0)})
		catalog.downloadRestore(
			t,
			restoreRequest{TargetTime: new(timeline.at(-60)), StatusCode: http.StatusConflict},
		)
	})
}
