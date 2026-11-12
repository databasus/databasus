package restoring_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func TestPhysicalRestoreHistory(t *testing.T) {
	t.Parallel()

	t.Run("falls back to completed backups when the selected timeline history has not published", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full := catalog.seedRestoreFull(
			t,
			restoreFullSpec{StartLSN: lsnAt(2), StopLSN: lsnAt(3), CompletedAt: timeline.at(0)},
		)
		wal := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 3, ReceivedAt: timeline.at(5)})
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET timeline_id=2 WHERE id=?", full.ID)
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_segments SET timeline_id=2,wal_filename='000000020000000000000003' WHERE id=?",
			wal.ID,
		)

		entries := catalog.downloadRestore(t, restoreRequest{})
		assertRestoreArtifacts(t, entries, full.ID, nil)
		entries = catalog.downloadRestore(t, restoreRequest{BackupID: &full.ID})
		assertRestoreArtifacts(t, entries, full.ID, nil)
		catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(4)), StatusCode: http.StatusConflict})

		unrelated := catalog.App.InsertPhysicalWalHistoryFile(t, apptest.PhysicalWalHistoryFileSeed{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 3,
			FileName: "unrelated.history.zst", HistoryFilename: "00000003.history",
		})
		catalog.publish(t, unrelated.FileName, zstdBytes(t, []byte("1\t0/1000000\tancestor\n2\t0/2000000\tlater\n")))
		catalog.publish(t, unrelated.FileName+".metadata", []byte(`{"encryption":"NONE"}`))
		assertRestoreArtifacts(t, catalog.downloadRestore(t, restoreRequest{}), full.ID, nil)

		history := catalog.App.InsertPhysicalWalHistoryFile(t, apptest.PhysicalWalHistoryFileSeed{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 2,
			FileName: "selected.history.zst", HistoryFilename: "00000002.history",
		})
		historyBody := "1\t0/2000000\tno recovery target specified\n"
		catalog.publish(t, history.FileName, zstdBytes(t, []byte(historyBody)))
		catalog.publish(t, history.FileName+".metadata", []byte(`{"encryption":"NONE"}`))

		entries = catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(4))})
		assert.Equal(t, full.ID.String(), entries["full/backup_manifest"])
		assert.Contains(t, entries, "wal/000000020000000000000003.zst")
		assert.Equal(t, historyBody, entries["wal/00000002.history"])
		assert.NotContains(t, entries, "wal/00000003.history")
	})

	t.Run("rejects a targeted download when required history disappears after token issuance", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		full := catalog.seedRestoreFull(
			t,
			restoreFullSpec{StartLSN: lsnAt(2), StopLSN: lsnAt(3), CompletedAt: timeline.at(0)},
		)
		wal := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 3, ReceivedAt: timeline.at(5)})
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET timeline_id=2 WHERE id=?", full.ID)
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical_wal_segments SET timeline_id=2 WHERE id=?", wal.ID)
		history := catalog.App.InsertPhysicalWalHistoryFile(t, apptest.PhysicalWalHistoryFileSeed{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 2,
			FileName: "selected.history.zst", HistoryFilename: "00000002.history",
		})
		var token restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, new(timeline.at(4))),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &token,
			},
		)

		var latestToken restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &latestToken,
			},
		)

		catalog.App.ExecuteSQL(t, "DELETE FROM backups_physical_wal_history_files WHERE id=?", history.ID)

		response := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusConflict},
		)
		assert.Contains(t, string(response.Body), `"code":"target_time_unreachable"`)
		assert.NotContains(t, response.Headers.Get("Content-Type"), "application/x-tar")
		latest := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalDownloadRequest(latestToken.Token), StatusCode: http.StatusOK},
		)
		assertRestoreArtifacts(t, readTarEntries(t, latest.Body), full.ID, nil)
	})
}
