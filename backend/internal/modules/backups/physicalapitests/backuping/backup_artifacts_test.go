package backuping_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalBackupArtifacts(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"17", "18"} {
		t.Run(
			"publishes a full and two incremental artifacts with their manifests when PostgreSQL "+version+" is healthy",
			func(t *testing.T) {
				t.Parallel()

				team := newBackupTeam(t, backupTeamSpec{Image: "postgres:" + version, IsEncrypted: true})
				full := team.runBackup(t, "full")
				require.Equal(t, "COMPLETED", full.Status)
				assert.Equal(t, "FULL", full.Type)

				database, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
				require.NoError(t, err)
				require.NotNil(t, database.JSON200)
				require.NotNil(t, database.JSON200.LastBackupTime)
				assert.True(t, full.CreatedAt.Equal(*database.JSON200.LastBackupTime))
				team.assertArtifacts(t, full)

				first := team.runBackup(t, "incremental")
				require.Equal(t, "COMPLETED", first.Status)
				assert.Equal(t, "INCREMENTAL", first.Type)
				team.assertArtifacts(t, first)

				second := team.runBackup(t, "incremental")
				require.Equal(t, "COMPLETED", second.Status)
				team.assertArtifacts(t, second)
				team.assertNoRunningBackup(t)

				var incremental struct {
					RootFullBackupID uuid.UUID
					ParentBackupID   *uuid.UUID
				}
				team.App.QuerySQL(t, &incremental, "SELECT * FROM backups_physical WHERE id=?", second.ID)
				assert.Equal(t, full.ID, incremental.RootFullBackupID)
				require.NotNil(t, incremental.ParentBackupID)
				assert.Equal(t, first.ID, *incremental.ParentBackupID)
			},
		)
	}
	t.Run("publishes a full with a measured raw cluster size when only full backups are enabled", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{BackupType: apiclient.PhysicalBackupTypeFULL})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)
		team.assertArtifacts(t, full)

		var stored apptest.PhysicalFullBackupSeed
		team.App.QuerySQL(t, &stored, "SELECT * FROM backups_physical WHERE id=?", full.ID)
		require.NotNil(t, stored.RawSizeMb)
		assert.Positive(t, *stored.RawSizeMb)
		team.assertNoRunningBackup(t)
	})
}

func (team *backupTeam) assertArtifacts(t *testing.T, backup backupRow) {
	t.Helper()
	var artifacts []struct {
		FileName               *string
		ManifestFileName       *string
		Compression            string
		Encryption             string
		EncryptionSalt         *string
		EncryptionIv           *string
		ManifestEncryptionSalt *string
		ManifestEncryptionIv   *string
		BackupSizeMb           *float64
		BackupDurationMs       *int64
	}

	table := "backups_physical"
	team.App.QuerySQL(t, &artifacts, "SELECT * FROM "+table+" WHERE id=?", backup.ID)
	require.Len(t, artifacts, 1)

	artifact := artifacts[0]
	require.NotNil(t, artifact.FileName)
	require.NotNil(t, artifact.ManifestFileName)
	assert.True(t, team.App.IsLocalStorageFilePresent(t, *artifact.FileName))
	assert.True(t, team.App.IsLocalStorageFilePresent(t, *artifact.ManifestFileName))
	assert.Equal(t, "ZSTD", artifact.Compression)
	assert.NotEmpty(t, backup.StartLSN)
	assert.NotEmpty(t, backup.StopLSN)
	require.NotNil(t, artifact.BackupSizeMb)
	assert.Positive(t, *artifact.BackupSizeMb)
	require.NotNil(t, artifact.BackupDurationMs)
	assert.Positive(t, *artifact.BackupDurationMs)

	if artifact.Encryption == "ENCRYPTED" {
		require.NotNil(t, artifact.EncryptionSalt)
		require.NotNil(t, artifact.EncryptionIv)
		require.NotNil(t, artifact.ManifestEncryptionSalt)
		require.NotNil(t, artifact.ManifestEncryptionIv)
		assert.NotEqual(t, *artifact.EncryptionSalt, *artifact.ManifestEncryptionSalt)
		assert.NotEqual(t, *artifact.EncryptionIv, *artifact.ManifestEncryptionIv)
	}
}
