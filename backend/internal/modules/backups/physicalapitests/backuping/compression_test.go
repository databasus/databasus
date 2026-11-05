package backuping_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/toolstub"
)

func TestPhysicalBackupCompression(t *testing.T) {
	t.Parallel()

	t.Run("completes with gzip and retries zstd on the next backup when the source rejects zstd", func(t *testing.T) {
		t.Parallel()

		workingDirectory, err := os.Getwd()
		require.NoError(t, err)
		bundleRoot, err := clienttools.FindBundleRoot(workingDirectory)
		require.NoError(t, err)

		realClientTools := pgshared.ClientTools{Root: bundleRoot}
		realExecutable := realClientTools.GetExecutable(pgshared.Version17, pgshared.ExecutablePgBasebackup)

		stubClientTools := pgshared.ClientTools{Root: t.TempDir()}
		stub := toolstub.Write(t,
			stubClientTools.GetExecutable(pgshared.Version17, pgshared.ExecutablePgBasebackup),
			toolstub.Spec{Script: `for argument in "$@"; do
    if [ "$argument" = '--compress=server-zstd:5' ]; then
        echo 'pg_basebackup: error: server does not support compression with zstd' >&2
        exit 1
    fi
done
exec '` + strings.ReplaceAll(realExecutable, "'", "'\"'\"'") + `' "$@"`})

		team := newBackupTeam(t, backupTeamSpec{ClientToolsDir: stubClientTools.Root})
		for backupNumber := range 2 {
			full := team.runBackup(t, "full")
			require.Equal(t, "COMPLETED", full.Status)
			var compression string
			team.App.QuerySQL(t, &compression, "SELECT compression FROM backups_physical WHERE id=?", full.ID)
			assert.Equal(t, "GZIP", compression)
			invocations := stub.GetInvocations(t)
			require.Len(t, invocations, 2*(backupNumber+1))
			assert.Contains(t, invocations[2*backupNumber], "--compress=server-zstd:5")
			assert.Contains(t, invocations[2*backupNumber+1], "--compress=server-gzip:6")
		}
	})
}
