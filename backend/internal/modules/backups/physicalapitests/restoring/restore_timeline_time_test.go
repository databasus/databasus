package restoring_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestPhysicalRestoreTimelineTime(t *testing.T) {
	t.Parallel()

	t.Run(
		"refuses explicit PITR while allowing latest and backup recovery when transaction evidence is missing",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			timeline := newRestoreTimeline()
			full := catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: timeline.at(0)})
			segment := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 1, ReceivedAt: timeline.at(15)})
			catalog.App.ExecuteSQL(
				t,
				"UPDATE backups_physical_wal_segments SET last_transaction_at=NULL WHERE id=?",
				segment.ID,
			)

			catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(7)), StatusCode: http.StatusConflict})
			catalog.downloadRestore(t, restoreRequest{})
			catalog.downloadRestore(t, restoreRequest{BackupID: &full.ID})
		},
	)

	t.Run("refuses the promotion gap when older timeline WAL publishes late", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		timeline.base = timeline.base.Truncate(time.Microsecond)
		full := catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: timeline.at(0)})
		segment := newWalSeed(walSeedSpec{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
			WalFilename: "000000010000000000000001", StartLSN: lsnAt(1), EndLSN: lsnAt(2),
		})
		segment.ReceivedAt = timeline.at(15)
		segment.LastTransactionAt = new(timeline.at(5))
		catalog.publish(t, *segment.FileName, []byte("old timeline WAL"))
		catalog.insertWal(t, segment)

		for _, shouldCompleteReplacement := range []bool{false, true} {
			if shouldCompleteReplacement {
				replacement := catalog.seedRestoreFull(t, restoreFullSpec{
					StartLSN: lsnAt(2), StopLSN: lsnAt(3), CompletedAt: timeline.at(10),
				})
				catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET timeline_id=2 WHERE id=?", replacement.ID)
			}

			response := catalog.call(t, catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, new(timeline.at(7))),
				Authorization: "Bearer " + catalog.Owner.Token, StatusCode: http.StatusConflict,
			})

			require.Contains(t, string(response.Body), `"code":"target_time_unreachable"`)
			require.NotContains(t, string(response.Body), `"token"`)
			entries := catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(4))})
			require.Equal(t, full.ID.String(), entries["full/backup_manifest"])
		}
	})

	t.Run("rejects download when confirming WAL disappears but late received WAL remains", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		timeline.base = timeline.base.Truncate(time.Microsecond)
		catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: timeline.at(0)})
		lateSegment := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 1, ReceivedAt: timeline.at(15)})
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical_wal_segments SET last_transaction_at=? WHERE id=?",
			timeline.at(5), lateSegment.ID)
		confirmingSegment := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 2, ReceivedAt: timeline.at(10)})
		var token restoreTokenResponse
		catalog.call(t, catalogCall{
			Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, new(timeline.at(7))),
			Authorization: "Bearer " + catalog.Owner.Token, StatusCode: http.StatusCreated, Destination: &token,
		})
		require.NotEmpty(t, token.Token)

		catalog.App.ExecuteSQL(t, "DELETE FROM backups_physical_wal_segments WHERE id=?", confirmingSegment.ID)

		response := catalog.call(t, catalogCall{
			Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusConflict,
		})

		require.Contains(t, string(response.Body), `"code":"target_time_unreachable"`)
		require.Contains(t, response.Headers.Get("Content-Type"), "application/problem+json")
		require.NotContains(t, response.Headers.Get("Content-Type"), "application/x-tar")
		require.Empty(t, response.Headers.Get("Content-Disposition"))
	})

	t.Run("refuses a target equal to the last transaction with inclusive recovery", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		timeline.base = timeline.base.Truncate(time.Microsecond)
		catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: timeline.at(0)})
		segment := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 1, ReceivedAt: timeline.at(5)})
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_segments SET last_transaction_at=? WHERE id=?",
			timeline.at(5),
			segment.ID,
		)

		catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(5)), StatusCode: http.StatusConflict})
		catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(5).Add(-time.Microsecond))})
	})
}
