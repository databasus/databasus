package usecases_physical_postgresql_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	storages_bridge "databasus-backend/internal/features/storages"
	"databasus-backend/internal/modules/storages"
)

func Test_FullOnly_ProducesArtifactAndManifest(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)

	backuping_physical.CreateTestPhysicalBackuper(nil).MakeBackup(t.Context(), fixture.BackupID, false)

	postgresql_executor.WaitForBackupStatus(t, fixture.BackupID, physical_enums.PhysicalBackupTypeFull,
		physical_enums.PhysicalBackupStatusCompleted, nil, 3*time.Minute)

	finalRow, err := physical_repositories.GetFullBackupRepository().FindByID(fixture.BackupID)
	require.NoError(t, err)
	require.NotNil(t, finalRow)

	require.NotNil(t, finalRow.FileName, "FileName must be populated post-COMPLETED")
	assert.False(t, strings.HasSuffix(*finalRow.FileName, ".tar"),
		"FileName must be extension-less, got %q", *finalRow.FileName)
	assert.False(t, strings.HasSuffix(*finalRow.FileName, ".zst"),
		"FileName must be extension-less, got %q", *finalRow.FileName)
	assert.Equal(t, physical_enums.PhysicalBackupCompressionZstd, finalRow.Compression,
		"the codec is recorded on the row, not the object name")

	require.NotNil(t, finalRow.StartLSN, "StartLSN must be populated")
	require.NotNil(t, finalRow.StopLSN, "StopLSN must be populated")
	require.NotNil(t, finalRow.BackupSizeMb, "BackupSizeMb must be populated")
	require.NotNil(t, finalRow.CompletedAt, "CompletedAt must be populated")

	assert.GreaterOrEqual(t, *finalRow.BackupSizeMb, float64(0))

	artifactReader, err := storages_bridge.OpenTestFile(
		t.Context(),
		storages.StoredFileReference{StorageID: fixture.Storage.ID, FileName: *finalRow.FileName},
	)
	require.NoError(t, err, "artifact %q must be in storage", *finalRow.FileName)
	require.NoError(t, artifactReader.Close())

	sidecarReader, err := storages_bridge.OpenTestFile(
		t.Context(),
		storages.StoredFileReference{StorageID: fixture.Storage.ID, FileName: *finalRow.FileName + ".metadata"},
	)
	require.NoError(t, err, "sidecar must be in storage")
	require.NoError(t, sidecarReader.Close())

	require.NotNil(t, finalRow.ManifestFileName, "ManifestFileName must be populated post-COMPLETED")
	assert.Equal(t, *finalRow.FileName+".manifest", *finalRow.ManifestFileName)

	manifestReader, err := storages_bridge.OpenTestFile(
		t.Context(),
		storages.StoredFileReference{StorageID: fixture.Storage.ID, FileName: *finalRow.ManifestFileName},
	)
	require.NoError(t, err, "manifest sidecar must be in storage")
	require.NoError(t, manifestReader.Close())
}
