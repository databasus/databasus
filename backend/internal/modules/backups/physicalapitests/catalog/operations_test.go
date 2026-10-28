package catalog_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func TestChangePhysicalBackup(t *testing.T) {
	t.Run("cancels and releases the claim when a backup is in progress", func(t *testing.T) {
		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)
		catalog.App.InsertPhysicalInFlightBackup(
			t,
			apptest.PhysicalInFlightBackupSeed{DatabaseID: catalog.DatabaseID, BackupID: full.ID, BackupType: "FULL"},
		)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String() + "/cancel",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          nil,
				StatusCode:    http.StatusNoContent,
			},
		)

		claim := catalog.getInFlightClaim(t)
		assert.Nil(t, claim, "cancel must release the in-flight claim")
	})
	t.Run("returns 400 when a completed backup is cancelled", func(t *testing.T) {
		catalog := newCatalog(t)

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

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String() + "/cancel",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          nil,
				StatusCode:    http.StatusBadRequest,
			},
		)
	})
	t.Run("removes its entire chain when a full backup is deleted", func(t *testing.T) {
		catalog := newCatalog(t)

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
					WalFilename: "000000010000000000000001",
					StartLSN:    fixtureLSN(segmentBytes),
					EndLSN:      fixtureLSN(2 * segmentBytes),
				},
			),
		)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodDelete,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String(),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertFullGone(t, full.ID)
		catalog.assertIncrementalGone(t, incr.ID)
		catalog.assertWalGone(t, walSegment.ID)
	})
	t.Run("keeps WAL and the root full when an incremental and its descendants are deleted", func(t *testing.T) {
		catalog := newCatalog(t)

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
		firstIncr := catalog.insertIncremental(
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
		secondIncr := catalog.insertIncremental(
			t,
			newIncrementalSeed(
				incrementalSeedSpec{
					DatabaseID:                catalog.DatabaseID,
					StorageID:                 catalog.StorageID,
					RootFullBackupID:          full.ID,
					ParentIncrementalBackupID: &firstIncr.ID,
					TimelineID:                1,
					StartLSN:                  fixtureLSN(2 * segmentBytes),
					StopLSN:                   fixtureLSN(3 * segmentBytes),
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
					WalFilename: "000000010000000000000001",
					StartLSN:    fixtureLSN(segmentBytes),
					EndLSN:      fixtureLSN(2 * segmentBytes),
				},
			),
		)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodDelete,
				Path:          "/api/v1/backups/physical/backups/" + firstIncr.ID.String(),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertIncrementalGone(t, firstIncr.ID)
		catalog.assertIncrementalGone(t, secondIncr.ID)
		catalog.assertFullPresent(t, full.ID)
		catalog.assertWalPresent(t, walSegment.ID)
	})
	t.Run("keeps the next anchor and its WAL when an older WAL segment is deleted", func(t *testing.T) {
		catalog := newCatalog(t)

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

		firstWal := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000001",
					StartLSN:    fixtureLSN(segmentBytes),
					EndLSN:      fixtureLSN(2 * segmentBytes),
				},
			),
		)
		secondWal := catalog.insertWal(
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
		thirdWal := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000003",
					StartLSN:    fixtureLSN(3 * segmentBytes),
					EndLSN:      fixtureLSN(4 * segmentBytes),
				},
			),
		)

		anchorIncr := catalog.insertIncremental(
			t,
			newIncrementalSeed(
				incrementalSeedSpec{
					DatabaseID:                catalog.DatabaseID,
					StorageID:                 catalog.StorageID,
					RootFullBackupID:          full.ID,
					ParentIncrementalBackupID: nil,
					TimelineID:                1,
					StartLSN:                  fixtureLSN(3 * segmentBytes),
					StopLSN:                   fixtureLSN(4 * segmentBytes),
				},
			),
		)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodDelete,
				Path:          "/api/v1/backups/physical/backups/" + firstWal.ID.String(),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertWalGone(t, firstWal.ID)
		catalog.assertWalGone(t, secondWal.ID)
		catalog.assertWalPresent(t, thirdWal.ID)
		catalog.assertFullPresent(t, full.ID)
		catalog.assertIncrementalPresent(t, anchorIncr.ID)
	})
	t.Run("keeps a backup and returns 400 when a viewer requests deletion", func(t *testing.T) {
		catalog := newCatalog(t)

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

		viewer := catalog.addViewer(t)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodDelete,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String(),
				Authorization: "Bearer " + viewer.Token,
				StatusCode:    http.StatusBadRequest,
			},
		)

		catalog.assertFullPresent(t, full.ID)
	})
	t.Run("keeps the claim and returns 400 when a viewer requests cancellation", func(t *testing.T) {
		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)
		catalog.App.InsertPhysicalInFlightBackup(
			t,
			apptest.PhysicalInFlightBackupSeed{DatabaseID: catalog.DatabaseID, BackupID: full.ID, BackupType: "FULL"},
		)

		viewer := catalog.addViewer(t)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String() + "/cancel",
				Authorization: "Bearer " + viewer.Token,
				Body:          nil,
				StatusCode:    http.StatusBadRequest,
			},
		)

		claim := catalog.getInFlightClaim(t)
		assert.NotNil(t, claim, "a viewer's rejected cancel must not release the claim")
	})
	t.Run("records the database name in the audit log when a full backup is deleted", func(t *testing.T) {
		catalog := newCatalog(t)

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

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodDelete,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String(),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertAuditLogContains(t, "Physical full backup deleted", catalog.DatabaseName)
	})
	t.Run("records the database name in the audit log when a backup is cancelled", func(t *testing.T) {
		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)
		catalog.App.InsertPhysicalInFlightBackup(
			t,
			apptest.PhysicalInFlightBackupSeed{DatabaseID: catalog.DatabaseID, BackupID: full.ID, BackupType: "FULL"},
		)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String() + "/cancel",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          nil,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertAuditLogContains(t, "Physical backup cancelled", catalog.DatabaseName)
	})
	t.Run("returns 404 when deletion names an unknown backup", func(t *testing.T) {
		catalog := newCatalog(t)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodDelete,
				Path:          "/api/v1/backups/physical/backups/" + uuid.NewString(),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("returns 404 when cancellation names an unknown backup", func(t *testing.T) {
		catalog := newCatalog(t)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/api/v1/backups/physical/backups/" + uuid.NewString() + "/cancel",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          nil,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("returns 404 when cancellation names a WAL segment", func(t *testing.T) {
		catalog := newCatalog(t)

		walSegment := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000001",
					StartLSN:    fixtureLSN(segmentBytes),
					EndLSN:      fixtureLSN(2 * segmentBytes),
				},
			),
		)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/api/v1/backups/physical/backups/" + walSegment.ID.String() + "/cancel",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          nil,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("cancels and releases the claim when an in progress backup is deleted", func(t *testing.T) {
		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)
		catalog.App.InsertPhysicalInFlightBackup(
			t,
			apptest.PhysicalInFlightBackupSeed{DatabaseID: catalog.DatabaseID, BackupID: full.ID, BackupType: "FULL"},
		)

		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodDelete,
				Path:          "/api/v1/backups/physical/backups/" + full.ID.String(),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertFullGone(t, full.ID)

		claim := catalog.getInFlightClaim(t)
		assert.Nil(t, claim, "deleting an in-progress backup must release its claim")
	})
}
