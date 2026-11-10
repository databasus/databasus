package catalog_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/storages"
)

func catalogLSN(segments int) fixtureLSN { return fixtureLSN(segments * segmentBytes) }

type cascadeFullSpec struct {
	Segment    int
	TimelineID int
	Offset     fixtureLSN
	Age        time.Duration
}

func (catalog *catalogFixture) seedCascadeFull(t *testing.T, spec cascadeFullSpec) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	timelineID := spec.TimelineID
	if timelineID == 0 {
		timelineID = 1
	}

	start := catalogLSN(spec.Segment) + spec.Offset
	full := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: timelineID,
			StartLSN:   start,
			StopLSN:    start + fixtureLSN(segmentBytes),
		},
	)

	full.CreatedAt = time.Now().UTC().Add(-spec.Age)
	full.CompletedAt = new(full.CreatedAt.Add(time.Minute))
	full.ManifestFileName = new(*full.FileName + ".manifest")
	return catalog.insertFull(t, full)
}

type cascadeWalSpec struct {
	Segment    int
	TimelineID int
	Age        time.Duration
}

func (catalog *catalogFixture) seedCascadeWal(t *testing.T, spec cascadeWalSpec) *apptest.PhysicalWalSegmentSeed {
	t.Helper()
	timelineID := spec.TimelineID
	if timelineID == 0 {
		timelineID = 1
	}

	wal := newWalSeed(
		walSeedSpec{
			DatabaseID:  catalog.DatabaseID,
			StorageID:   catalog.StorageID,
			TimelineID:  timelineID,
			WalFilename: fmtWalName(timelineID, spec.Segment),
			StartLSN:    catalogLSN(spec.Segment),
			EndLSN:      catalogLSN(spec.Segment + 1),
		},
	)
	wal.ReceivedAt = time.Now().UTC().Add(-spec.Age)
	return catalog.insertWal(t, wal)
}

func (catalog *catalogFixture) seedCascadeIncremental(
	t *testing.T,
	fullID uuid.UUID,
	segment int,
) *apptest.PhysicalIncrementalBackupSeed {
	t.Helper()
	incremental := newIncrementalSeed(
		incrementalSeedSpec{
			DatabaseID:       catalog.DatabaseID,
			StorageID:        catalog.StorageID,
			RootFullBackupID: fullID,
			TimelineID:       1,
			StartLSN:         catalogLSN(segment),
			StopLSN:          catalogLSN(segment + 1),
		},
	)
	incremental.ManifestFileName = new(*incremental.FileName + ".manifest")
	return catalog.insertIncremental(t, incremental)
}

func (catalog *catalogFixture) deleteFull(t *testing.T, backupID uuid.UUID) {
	t.Helper()
	catalog.call(
		t,
		catalogCall{
			Request:       apptest.PhysicalDeleteRequest(backupID),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    http.StatusNoContent,
		},
	)
}

func (catalog *catalogFixture) publishNames(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		catalog.App.PublishStorageFile(
			t,
			storages.StoredFileReference{StorageID: catalog.StorageID, FileName: name},
			strings.NewReader("stored artifact"),
		)
	}
}

func (catalog *catalogFixture) assertFilesDeleted(t *testing.T, names ...string) {
	t.Helper()
	for range (len(names) + 9) / 10 {
		catalog.App.RunJobPass(t, "storage_file_deletion")
	}
	for _, name := range names {
		assert.False(t, catalog.App.IsLocalStorageFilePresent(t, name), "%s", name)
	}
}

func (catalog *catalogFixture) seedHistory(t *testing.T, timelineID int) apptest.PhysicalWalHistoryFileSeed {
	t.Helper()
	return catalog.App.InsertPhysicalWalHistoryFile(
		t,
		apptest.PhysicalWalHistoryFileSeed{
			DatabaseID:      catalog.DatabaseID,
			StorageID:       catalog.StorageID,
			TimelineID:      timelineID,
			FileName:        uuid.NewString() + ".history.zst",
			HistoryFilename: fmtHistoryName(timelineID),
		},
	)
}

func (catalog *catalogFixture) assertHistoryPresent(t *testing.T, historyID uuid.UUID, isPresent bool) {
	t.Helper()
	var count int64
	catalog.App.QuerySQL(t, &count, "SELECT count(*) FROM backups_physical_wal_history_files WHERE id=?", historyID)
	assert.Equal(t, isPresent, count == 1)
}

func (catalog *catalogFixture) keepFullPass(t *testing.T) {
	t.Helper()
	config := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
	settings, err := config.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	configureRetentionSchedule(&settings)
	settings.Retention.FullBackups = &apiclient.FullBackupsRetention{
		Policy: apiclient.FullBackupsRetentionPolicyLASTN,
		Count:  20,
	}

	cleaner := cleanerCatalog{catalogFixture: catalog, Config: settings}
	cleaner.clean(t, apiclient.PhysicalRetentionPolicyFULLBACKUPS)
}

func fmtWalName(timelineID, segment int) string { return fmt.Sprintf("%08X%016X", timelineID, segment) }
func fmtHistoryName(timelineID int) string      { return fmt.Sprintf("%08X.history", timelineID) }
