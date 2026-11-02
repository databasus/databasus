package walstreaming_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestArchivePhysicalWal(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"17", "18"} {
		t.Run(
			"archives complete segments and records the backup time when PostgreSQL "+version+" streams WAL",
			func(t *testing.T) {
				t.Parallel()

				team := newWalTeam(t, walTeamSpec{Image: "postgres:" + version})
				team.enable(t)
				team.waitForReceiver(t)

				source := team.openSource(t)
				for range 3 {
					rotateWal(t, source)
				}
				team.waitForWal(t, 1)

				var artifacts []struct {
					FileName         string
					DatabaseID       string
					CompressedSizeMb float64
				}
				team.App.QuerySQL(
					t,
					&artifacts,
					"SELECT file_name,database_id,compressed_size_mb FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL",
					team.Database.Id,
				)
				require.NotEmpty(t, artifacts)
				for _, artifact := range artifacts {
					assert.Equal(t, team.Database.Id.String(), artifact.DatabaseID)
					assert.True(t, team.App.IsLocalStorageFilePresent(t, artifact.FileName))
					assert.Positive(t, artifact.CompressedSizeMb)
				}

				database, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
				require.NoError(t, err)
				require.NotNil(t, database.JSON200)
				assert.NotNil(t, database.JSON200.LastBackupTime)

				testwait.Until(t, "the live route reports archive progress", func(context.Context) error {
					reading := team.live(t)
					if reading.Current == nil || reading.Current.ArchiveProgressLSN == nil ||
						reading.Current.LastArchivedWalAt == nil {
						return fmt.Errorf("no live archive position")
					}
					return nil
				})
			},
		)
	}
}

func TestArchivePhysicalWalAfterRestart(t *testing.T) {
	t.Parallel()

	t.Run("starts a new session and keeps its slot and queue when modules restart", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		rotateWal(t, source)
		team.waitForWal(t, 1)

		var firstStarted string
		team.App.QuerySQL(
			t,
			&firstStarted,
			"SELECT started_at::text FROM backups_physical_wal_streamers WHERE database_id=?",
			team.Database.Id,
		)
		var slot string
		team.App.QuerySQL(
			t,
			&slot,
			"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
			team.Database.Id,
		)
		assert.True(t, team.App.IsWalQueueDirectoryPresent(t, team.Database.Id))

		team.App.RestartModules(t)
		team.waitForReceiver(t)
		testwait.Until(t, "the streamer has a new session", func(context.Context) error {
			var started string
			team.App.QuerySQL(
				t,
				&started,
				"SELECT started_at::text FROM backups_physical_wal_streamers WHERE database_id=?",
				team.Database.Id,
			)
			if started == firstStarted {
				return fmt.Errorf("the old session remains")
			}
			return nil
		})

		var count int
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slot).
				Scan(&count),
		)
		assert.Equal(t, 1, count)

		rotateWal(t, source)
		team.waitForWal(t, 2)
		assert.True(t, team.App.IsWalQueueDirectoryPresent(t, team.Database.Id))
	})
}
