package logical

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/toolstub"
)

const publicSchemaClashStderr = `pg_restore: error: could not execute query: ERROR:  schema "public" already exists
Command was: CREATE SCHEMA public;


pg_restore: warning: errors ignored on restore: 1
`

const pgRestoreStubError = `pg_restore: error: could not execute query: ERROR:  relation "orders" already exists`

// copyLastArgumentScript copies the file named by the stub's last argument, the archive, to the
// path that follows, so a test sees what pg_restore would have read.
const copyLastArgumentScript = `for archive in "$@"; do :; done; cp "$archive" `

func TestRestore(t *testing.T) {
	t.Parallel()

	t.Run("streams the archive into pg_restore with a single job", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		pgRestore := writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: true,
			Script:         "",
		})
		spec := newStubRestoreSpec(t, clientTools)

		err := newStubSource("orders_restored").Restore(t.Context(), spec, bytes.NewReader([]byte("PGDMP-archive")))

		require.NoError(t, err)
		assert.Equal(t, "PGDMP-archive", string(pgRestore.GetStdin(t)))
		assert.Equal(t, []string{
			"-Fc", "--no-password",
			"-h", "db.example.com", "-p", "5432", "-U", "backup_user",
			"-d", "orders_restored",
			"--verbose",
			"--clean", "--if-exists",
			"--no-owner", "--no-acl",
		}, pgRestore.GetLastArgs(t))
	})

	t.Run("restores from a file with parallel jobs capped at 8 and removes the file", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		restoredArchivePath := filepath.Join(t.TempDir(), "restored-archive")
		pgRestore := writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: false,
			Script:         copyLastArgumentScript + "'" + restoredArchivePath + "'",
		})
		spec := newStubRestoreSpec(t, clientTools)
		spec.ParallelJobs = 16
		spec.IsRestoreOwnership = true
		spec.IsRestorePrivileges = true

		err := newStubSource("orders_restored").Restore(t.Context(), spec, bytes.NewReader([]byte("PGDMP-archive")))

		require.NoError(t, err)
		restoredArchive, err := os.ReadFile(restoredArchivePath)
		require.NoError(t, err)
		assert.Equal(t, "PGDMP-archive", string(restoredArchive))
		restoreArgs := pgRestore.GetLastArgs(t)
		assert.Equal(t, []string{
			"-Fc", "-j", "8", "--no-password",
			"-h", "db.example.com", "-p", "5432", "-U", "backup_user",
			"-d", "orders_restored",
			"--verbose",
			"--clean", "--if-exists",
		}, restoreArgs[:len(restoreArgs)-1])
		assert.Empty(t, readDirNames(t, spec.TempDir))
	})

	t.Run("restores a filtered table of contents when user mappings are skipped", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		restoredTocPath := filepath.Join(t.TempDir(), "restored-toc")
		pgRestore := writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: false,
			Script: `if [ "$1" = "-l" ]; then
  printf '3420; 0 0 EXTENSION - uuid-ossp\n18239; 0 0 USER MAPPING - platform SERVER oracle_tb_server\n215; 1259 16390 TABLE public orders\n'
  exit 0
fi
while [ $# -gt 0 ]; do
  if [ "$1" = "-L" ]; then cp "$2" '` + restoredTocPath + `'; fi
  shift
done`,
		})
		spec := newStubRestoreSpec(t, clientTools)
		spec.IsSkipUserMappings = true

		err := newStubSource("orders_restored").Restore(t.Context(), spec, bytes.NewReader([]byte("PGDMP-archive")))

		require.NoError(t, err)
		invocations := pgRestore.GetInvocations(t)
		require.Len(t, invocations, 2)
		assert.Equal(t, "-l", invocations[0][0])
		assert.Contains(t, invocations[1], "-L")
		restoredToc, err := os.ReadFile(restoredTocPath)
		require.NoError(t, err)
		assert.Equal(t,
			"3420; 0 0 EXTENSION - uuid-ossp\n215; 1259 16390 TABLE public orders",
			string(restoredToc),
		)
	})

	t.Run("succeeds when the only error is the existing public schema", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: true,
			Script:         "printf '%s' '" + publicSchemaClashStderr + "' >&2; exit 1",
		})

		err := newStubSource("orders_restored").Restore(
			t.Context(), newStubRestoreSpec(t, clientTools), bytes.NewReader([]byte("PGDMP")),
		)

		require.NoError(t, err)
	})

	t.Run("fails when the target cannot create an extension", func(t *testing.T) {
		t.Parallel()
		// Verification may lack source extensions, but a real restore must not lose their data.
		stderr := `pg_restore: error: could not execute query: ERROR: extension "postgis" is not available
Command was: CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;
pg_restore: error: could not execute query: ERROR: extension "postgis" does not exist
Command was: COMMENT ON EXTENSION postgis IS NULL;
pg_restore: warning: errors ignored on restore: 2
`
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: true,
			Script:         "printf '%s' '" + stderr + "' >&2; exit 1",
		})

		err := newStubSource("orders_restored").Restore(
			t.Context(), newStubRestoreSpec(t, clientTools), bytes.NewReader([]byte("PGDMP")),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `extension "postgis" is not available`)
	})

	t.Run("reports the pg_restore error written after pg_restore exited", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: true,
			Script:         "(sleep 0.2; echo '" + pgRestoreStubError + "' >&2) &\nexit 1",
		})

		err := newStubSource("orders_restored").Restore(
			t.Context(), newStubRestoreSpec(t, clientTools), bytes.NewReader([]byte("PGDMP")),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `relation "orders" already exists`)
	})

	t.Run("reports the pg_restore error when pg_restore exits before reading the archive", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: false,
			Script:         "echo '" + pgRestoreStubError + "' >&2; exit 1",
		})
		largerThanPipeBuffer := bytes.Repeat([]byte("PGDMP"), 4*1024*1024)

		err := newStubSource("orders_restored").Restore(
			t.Context(), newStubRestoreSpec(t, clientTools), bytes.NewReader(largerThanPipeBuffer),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `relation "orders" already exists`)
	})

	t.Run("names both databases when the target database does not exist", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgRestore, toolstub.Spec{
			IsReadingStdin: true,
			Script:         `echo 'FATAL:  database "orders_restored" does not exist' >&2; exit 1`,
		})
		spec := newStubRestoreSpec(t, clientTools)
		spec.SourceDatabaseName = "orders"

		err := newStubSource("orders_restored").Restore(t.Context(), spec, bytes.NewReader([]byte("PGDMP")))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Target database does not exist (backup db orders, not found orders_restored)")
	})

	t.Run("refuses a target without a database name", func(t *testing.T) {
		t.Parallel()

		err := newStubSource("").Restore(
			t.Context(),
			newStubRestoreSpec(t, pgshared.ClientTools{Root: t.TempDir()}),
			bytes.NewReader([]byte("PGDMP")),
		)

		require.ErrorIs(t, err, ErrRestoreDatabaseNameRequired)
	})
}

func newStubRestoreSpec(t *testing.T, clientTools pgshared.ClientTools) RestoreSpec {
	t.Helper()

	return RestoreSpec{
		NewCleanupContext:   nil,
		ClientTools:         clientTools,
		Version:             pgshared.Version16,
		SourceDatabaseName:  "",
		ParallelJobs:        1,
		IsTimescaleDB:       false,
		IsRestoreOwnership:  false,
		IsRestorePrivileges: false,
		IsExcludeExtensions: false,
		IsSkipUserMappings:  false,
		TempDir:             t.TempDir(),
		Logger:              slog.New(slog.DiscardHandler),
	}
}

func readDirNames(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names
}
