package backuping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestPhysicalBackupWithoutClusterSize(t *testing.T) {
	t.Parallel()

	t.Run("completes with a null raw size when the role cannot measure database sizes", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		team.useReplicationRole(t)
		source := team.openSource(t)
		_, err := source.Exec(t.Context(), "REVOKE EXECUTE ON FUNCTION pg_database_size(name) FROM PUBLIC")
		require.NoError(t, err)

		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)

		var stored apptest.PhysicalFullBackupSeed
		team.App.QuerySQL(t, &stored, "SELECT * FROM backups_physical WHERE id=?", full.ID)
		assert.Nil(t, stored.RawSizeMb)
		team.assertArtifacts(t, full)
	})
}
