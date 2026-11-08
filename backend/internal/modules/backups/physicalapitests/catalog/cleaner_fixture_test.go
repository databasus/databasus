package catalog_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

type cleanerCatalog struct {
	*catalogFixture
	Config          apiclient.PhysicalBackupSettings
	hasRecentChains bool
}

func newCleanerCatalog(t *testing.T) *cleanerCatalog {
	t.Helper()
	catalog := newCatalog(t)
	readConfig := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
	settings, err := readConfig.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	return &cleanerCatalog{catalogFixture: catalog, Config: settings}
}

func configureRetentionSchedule(settings *apiclient.PhysicalBackupSettings) {
	settings.FullBackupSchedule = apiclient.Schedule{Type: apiclient.ScheduleTypeDAILY, TimeOfDay: new("04:00")}
	settings.IncrementalBackupSchedule = &apiclient.Schedule{Type: apiclient.ScheduleTypeHOURLY}
}

func (catalog *cleanerCatalog) clean(t *testing.T, policy apiclient.PhysicalRetentionPolicy) {
	t.Helper()
	settings := catalog.Config
	settings.Retention.Policy = policy

	switch policy {
	case apiclient.PhysicalRetentionPolicyCHAINS:
		settings.Retention.FullBackups = nil
	case apiclient.PhysicalRetentionPolicyFULLBACKUPS:
		catalog.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET backup_type = 'FULL' WHERE database_id = ?",
			catalog.DatabaseID,
		)
		settings.IncrementalBackupSchedule = nil
		settings.WalLagThresholdBytes = 0
		settings.Retention.ChainsCount = 0
	case apiclient.PhysicalRetentionPolicyCHAINSANDFULLBACKUPS:
	}

	var backupSettings apiclient.BackupSettings
	require.NoError(t, backupSettings.FromPhysicalBackupSettings(settings))
	response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		catalog.DatabaseID,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &catalog.StorageID,
			Encryption:       apiclient.BackupEncryptionNONE,
			Settings:         backupSettings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)

	catalog.App.RunJobPass(t, "backups_physical_cleaner")
}

func cleanerLSN(segment int) fixtureLSN { return fixtureLSN(segment * segmentBytes) }

type chainFullSpec struct {
	StartSegment int
	AgeHours     int
}

func (catalog *cleanerCatalog) seedChainFull(t *testing.T, spec chainFullSpec) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	full := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: 1,
			StartLSN:   cleanerLSN(spec.StartSegment),
			StopLSN:    cleanerLSN(spec.StartSegment + 1),
		},
	)

	age := time.Duration(spec.AgeHours) * time.Hour
	if !catalog.hasRecentChains {
		age += 48*time.Hour + 10*time.Minute
	}
	timestamp := time.Now().UTC().Add(-age)
	full.CreatedAt = timestamp
	full.CompletedAt = &timestamp
	full.BackupSizeMb = new(1000.0)

	return catalog.insertFull(t, full)
}

type agedFullSpec struct {
	Status       string
	StartSegment int
	AgeHours     int
}

func (catalog *cleanerCatalog) seedFullWithStatusAndAge(
	t *testing.T,
	spec agedFullSpec,
) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	full := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: 1,
			StartLSN:   cleanerLSN(spec.StartSegment),
			StopLSN:    cleanerLSN(spec.StartSegment + 1),
		},
	)
	full.CreatedAt = time.Now().UTC().Add(-time.Duration(spec.AgeHours) * time.Hour)
	full.Status = spec.Status
	return catalog.insertFull(t, full)
}

func (catalog *cleanerCatalog) fullExists(t *testing.T, backupID uuid.UUID) bool {
	t.Helper()
	return slices.Contains(collectBackupIDs(getBackups(t, catalog.catalogFixture, "?type=FULL&limit=1000")), backupID)
}

func (catalog *cleanerCatalog) incrementalExists(t *testing.T, backupID uuid.UUID) bool {
	t.Helper()
	return slices.Contains(
		collectBackupIDs(getBackups(t, catalog.catalogFixture, "?type=INCREMENTAL&limit=1000")),
		backupID,
	)
}

func (catalog *cleanerCatalog) walExists(t *testing.T, segmentID uuid.UUID) bool {
	t.Helper()
	var count int64
	catalog.App.QuerySQL(t, &count, "SELECT count(*) FROM backups_physical_wal_segments WHERE id = ?", segmentID)
	return count > 0
}

func newCleanerSample(databaseID uuid.UUID, sampledAt time.Time) *apptest.PhysicalWalStreamSampleSeed {
	return &apptest.PhysicalWalStreamSampleSeed{DatabaseID: databaseID, SampledAt: sampledAt}
}

func (catalog *cleanerCatalog) insertCleanerSample(
	t *testing.T,
	sample *apptest.PhysicalWalStreamSampleSeed,
) *apptest.PhysicalWalStreamSampleSeed {
	t.Helper()
	*sample = catalog.App.InsertPhysicalWalStreamSample(t, *sample)
	return sample
}

type datedFullSpec struct {
	CreatedAt    time.Time
	StartSegment int
}

func (catalog *cleanerCatalog) seedCompletedFullAt(t *testing.T, spec datedFullSpec) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	full := newFullSeed(fullSeedSpec{
		DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
		StartLSN: cleanerLSN(spec.StartSegment), StopLSN: cleanerLSN(spec.StartSegment + 1),
	})
	full.CreatedAt = spec.CreatedAt
	full.CompletedAt = &spec.CreatedAt

	return catalog.insertFull(t, full)
}
