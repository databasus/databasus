package catalog_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestCancelPhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run("returns the backup in its final status when a full backup is running", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		runningFull := catalog.insertFull(t, newInProgressFull(inProgressFullSpec{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
		}))

		cancelledBackup := catalog.cancelBackup(t, apptest.PhysicalCancelRequest(runningFull.ID))

		assert.Equal(t, runningFull.ID, cancelledBackup.ID)
		assert.Equal(t, "FULL", cancelledBackup.Type)
		assert.Equal(t, "CANCELED", cancelledBackup.Status)
		assert.NotNil(t, cancelledBackup.CompletedAt)

		listedBackups := getBackups(t, catalog, "").Backups
		require.Len(t, listedBackups, 1)
		assert.Equal(t, listedBackups[0], cancelledBackup)
	})
	t.Run("returns the backup in its final status when an incremental on the root is running", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		rootFull := catalog.insertFull(t, newFullSeed(fullSeedSpec{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
			StartLSN: fixtureLSN(0), StopLSN: fixtureLSN(segmentBytes),
		}))
		incrementalSeed := newIncrementalSeed(incrementalSeedSpec{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, RootFullBackupID: rootFull.ID,
			TimelineID: 1, StartLSN: fixtureLSN(segmentBytes), StopLSN: fixtureLSN(2 * segmentBytes),
		})
		incrementalSeed.Status = "IN_PROGRESS"
		incrementalSeed.FileName = nil
		incrementalSeed.StartLSN = nil
		incrementalSeed.StopLSN = nil
		runningIncremental := catalog.insertIncremental(t, incrementalSeed)

		cancelledBackup := catalog.cancelBackup(t, apptest.PhysicalCancelRequest(runningIncremental.ID))

		assert.Equal(t, runningIncremental.ID, cancelledBackup.ID)
		assert.Equal(t, "INCREMENTAL", cancelledBackup.Type)
		assert.Equal(t, "CANCELED", cancelledBackup.Status)
		assert.Equal(t, &rootFull.ID, cancelledBackup.RootFullBackupID)
		assert.Nil(t, cancelledBackup.ParentIncrementalBackupID)

		listedIncrementals := getBackups(t, catalog, "?type=INCREMENTAL").Backups
		require.Len(t, listedIncrementals, 1)
		assert.Equal(t, listedIncrementals[0], cancelledBackup)
	})
}

func (catalog *catalogFixture) cancelBackup(t *testing.T, request apptest.PhysicalRequest) backupRow {
	t.Helper()

	var cancelledBackup backupRow
	catalog.call(t, catalogCall{
		Request:       request,
		Authorization: "Bearer " + catalog.Owner.Token,
		StatusCode:    http.StatusOK,
		Destination:   &cancelledBackup,
	})

	return cancelledBackup
}
