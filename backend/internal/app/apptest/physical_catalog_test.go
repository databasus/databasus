package apptest

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhysicalCatalogSeeds(t *testing.T) {
	t.Parallel()

	t.Run("lists catalog rows when a physical database has no source server", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		owner := testApp.CreateAdmin(t)
		workspace := CreateWorkspace(t, owner, "Catalog seeds")
		storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)
		databaseID := testApp.InsertPhysicalDatabase(t, workspace.Id)
		testApp.InsertPhysicalBackupConfigUsingStorage(t, StorageUse{DatabaseID: databaseID, StorageID: storageID})
		full := testApp.InsertPhysicalFullBackup(t, PhysicalFullBackupSeed{
			DatabaseID: databaseID, StorageID: storageID, StartLSN: new("0/1000028"), StopLSN: new("0/1000128"),
		})
		incremental := testApp.InsertPhysicalIncrementalBackup(t, PhysicalIncrementalBackupSeed{
			DatabaseID: databaseID, StorageID: storageID, RootFullBackupID: full.ID,
			StartLSN: new("0/2000028"), StopLSN: new("0/2000128"),
		})
		wal := testApp.InsertPhysicalWalSegment(t, PhysicalWalSegmentSeed{
			DatabaseID: databaseID, StorageID: storageID, FileName: new("seed-wal"),
			WalFilename: "000000010000000000000002", StartLSN: "0/2000000", EndLSN: "0/3000000",
		})
		testApp.InsertPhysicalWalHistoryFile(t, PhysicalWalHistoryFileSeed{
			DatabaseID: databaseID, StorageID: storageID, TimelineID: 2,
			FileName: "seed-history", HistoryFilename: "00000002.history",
		})

		testApp.InsertPhysicalWalStreamer(t, PhysicalWalStreamerSeed{DatabaseID: databaseID})
		testApp.InsertPhysicalWalStreamSample(t, PhysicalWalStreamSampleSeed{
			DatabaseID: databaseID, WalSegmentSizeBytes: 16 * 1024 * 1024,
		})

		listedBackups := listPhysicalBackups(t, testApp, owner, databaseID)

		require.Len(t, listedBackups, 3)
		backupIDs := make([]uuid.UUID, 0, len(listedBackups))
		for _, backup := range listedBackups {
			backupIDs = append(backupIDs, backup.ID)
		}
		assert.ElementsMatch(t, []uuid.UUID{full.ID, incremental.ID, wal.ID}, backupIDs)
		statistics := getPhysicalStreamStatistics(t, testApp, owner, databaseID)
		require.NotNil(t, statistics.Current)
	})
}
