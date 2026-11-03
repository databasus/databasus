package catalog_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestChangePhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run("records cancellation before answering when a running backup has no live process", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalCancelRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusOK,
			},
		)

		runningBackupID := catalog.getRunningBackupID(t)
		assert.Nil(t, runningBackupID, "cancel must leave no running backup")
		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 1)
		assert.Equal(t, "CANCELED", page.Backups[0].Status)
	})
	t.Run("returns 409 when a completed backup is cancelled", func(t *testing.T) {
		t.Parallel()

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

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalCancelRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusConflict,
				Destination:   &problem,
			},
		)
		assert.Equal(t, apiclient.ProblemCodeNotRunning, problem.Code)
		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 1)
		assert.Equal(t, full.ID, page.Backups[0].ID)
		assert.Equal(t, "COMPLETED", page.Backups[0].Status)
	})
	t.Run("removes its entire chain when a full backup is deleted", func(t *testing.T) {
		t.Parallel()

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
				Request:       apptest.PhysicalDeleteRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertFullGone(t, full.ID)
		catalog.assertIncrementalGone(t, incr.ID)
		catalog.assertWalGone(t, walSegment.ID)
	})
	t.Run("keeps WAL and the root full when an incremental and its descendants are deleted", func(t *testing.T) {
		t.Parallel()

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
				Request:       apptest.PhysicalDeleteRequest(firstIncr.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertIncrementalGone(t, firstIncr.ID)
		catalog.assertIncrementalGone(t, secondIncr.ID)
		catalog.assertFullPresent(t, full.ID)
		catalog.assertWalPresent(t, walSegment.ID)
	})
	t.Run("keeps a backup and returns 403 when a viewer requests deletion", func(t *testing.T) {
		t.Parallel()

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

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalDeleteRequest(full.ID),
				Authorization: "Bearer " + viewer.Token,
				StatusCode:    http.StatusForbidden,
				Destination:   &problem,
			},
		)
		assert.Equal(t, apiclient.ProblemCodePermissionDenied, problem.Code)

		catalog.assertFullPresent(t, full.ID)
	})
	t.Run("keeps the backup running and returns 403 when a viewer requests cancellation", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		viewer := catalog.addViewer(t)

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalCancelRequest(full.ID),
				Authorization: "Bearer " + viewer.Token,
				StatusCode:    http.StatusForbidden,
				Destination:   &problem,
			},
		)
		assert.Equal(t, apiclient.ProblemCodePermissionDenied, problem.Code)

		runningBackupID := catalog.getRunningBackupID(t)
		assert.NotNil(t, runningBackupID, "a viewer's rejected cancel must leave the backup running")
	})
	t.Run("records the database name in the audit log when a full backup is deleted", func(t *testing.T) {
		t.Parallel()

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
				Request:       apptest.PhysicalDeleteRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertAuditLogContains(t, "Physical full backup deleted", catalog.DatabaseName)
	})
	t.Run("records the database name in the audit log when a backup is cancelled", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalCancelRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusOK,
			},
		)

		catalog.assertAuditLogContains(t, "Physical backup cancelled", catalog.DatabaseName)
	})
	t.Run("returns 404 when deletion names an unknown backup", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalDeleteRequest(uuid.New()),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("returns 404 when cancellation names an unknown backup", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalCancelRequest(uuid.New()),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("returns 404 when cancellation names a WAL segment", func(t *testing.T) {
		t.Parallel()

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
				Request:       apptest.PhysicalCancelRequest(walSegment.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("removes the running backup before answering when it is deleted", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalDeleteRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		catalog.assertFullGone(t, full.ID)

		runningBackupID := catalog.getRunningBackupID(t)
		assert.Nil(t, runningBackupID, "deleting a running backup must remove its row")
	})
}

func TestPhysicalBackupMemberChanges(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name     string
		isCancel bool
	}{
		{"cancels the running backup when the user is a workspace member", true},
		{"deletes a completed backup when the user is a workspace member", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			member := catalog.App.CreateNonAdminUser(t, catalog.Admin)
			apptest.AddWorkspaceMember(t, catalog.Owner, catalog.WorkspaceID, member.Email,
				apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			if scenario.isCancel {
				running := catalog.insertFull(t, newInProgressFull(inProgressFullSpec{
					DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
				}))

				catalog.call(t, catalogCall{
					Request:       apptest.PhysicalCancelRequest(running.ID),
					Authorization: "Bearer " + member.Token, StatusCode: http.StatusOK,
				})

				assert.Nil(t, catalog.getRunningBackupID(t))
				page := getBackups(t, catalog, "")
				require.Len(t, page.Backups, 1)
				assert.Equal(t, running.ID, page.Backups[0].ID)
				assert.Equal(t, "CANCELED", page.Backups[0].Status)
			} else {
				full := catalog.insertFull(t, newFullSeed(fullSeedSpec{
					DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
					StartLSN: fixtureLSN(0), StopLSN: fixtureLSN(segmentBytes),
				}))
				catalog.call(t, catalogCall{
					Request:       apptest.PhysicalDeleteRequest(full.ID),
					Authorization: "Bearer " + member.Token, StatusCode: http.StatusNoContent,
				})
				catalog.assertFullGone(t, full.ID)
			}
		})
	}
}
