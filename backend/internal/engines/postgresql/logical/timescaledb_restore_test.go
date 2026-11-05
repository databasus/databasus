package logical

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // The subtests share one TimescaleDB server.
func TestTimescaleRestore(t *testing.T) {
	t.Parallel()
	server := testServer{
		endpoint: containers.StartTimescaleDB(t, "timescale/timescaledb:2.17.0-pg17"),
		version:  pgshared.Version17,
	}

	t.Run("leaves restoring mode after cancellation", func(t *testing.T) {
		database := server.createDatabase(t)
		database.exec(t, "CREATE EXTENSION IF NOT EXISTS timescaledb")
		spec := newTimescaleRestoreSpec(t)

		restoreCtx, cancelRestore := context.WithCancel(t.Context())
		defer cancelRestore()
		archiveReader, archiveWriter := io.Pipe()
		defer func() { _ = archiveReader.Close() }()
		defer func() { _ = archiveWriter.Close() }()

		stopClosingArchive := context.AfterFunc(
			restoreCtx,
			func() { _ = archiveWriter.CloseWithError(restoreCtx.Err()) },
		)
		defer stopClosingArchive()

		restoreErrors := make(chan error, 1)
		go func() { restoreErrors <- database.newAdminSource().Restore(restoreCtx, spec, archiveReader) }()

		testwait.Until(t, "the target enters restoring mode", func(context.Context) error {
			if !database.isTimescaleRestoring(t) {
				return fmt.Errorf("restoring mode is not enabled")
			}
			return nil
		})
		cancelRestore()
		require.Error(t, <-restoreErrors)
		assert.False(t, database.isTimescaleRestoring(t))
	})

	t.Run("leaves restoring mode after an archive failure", func(t *testing.T) {
		database := server.createDatabase(t)
		spec := newTimescaleRestoreSpec(t)

		err := database.newAdminSource().Restore(t.Context(), spec, bytes.NewBufferString("invalid archive"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pg_restore")
		assert.False(t, database.isTimescaleRestoring(t))
	})

	t.Run("preserves cancellation and warns when the target refuses cleanup connections", func(t *testing.T) {
		database := server.createDatabase(t)
		spec := newTimescaleRestoreSpec(t)
		var restoreLog bytes.Buffer
		spec.Logger = slog.New(slog.NewTextHandler(&restoreLog, nil))

		restoreCtx, cancelRestore := context.WithCancel(t.Context())
		defer cancelRestore()
		archiveReader, archiveWriter := io.Pipe()
		defer func() { _ = archiveReader.Close(); _ = archiveWriter.Close() }()
		stopClosing := context.AfterFunc(restoreCtx, func() { _ = archiveWriter.CloseWithError(restoreCtx.Err()) })
		defer stopClosing()

		restoreErrors := make(chan error, 1)
		go func() { restoreErrors <- database.newAdminSource().Restore(restoreCtx, spec, archiveReader) }()
		testwait.Until(t, "the target enters restoring mode", func(context.Context) error {
			if !database.isTimescaleRestoring(t) {
				return fmt.Errorf("restoring mode is not enabled")
			}
			return nil
		})

		serverConnection := server.connectAs(t, getAdminRole(), containers.PostgresDatabase)
		_, err := serverConnection.Exec(
			t.Context(),
			fmt.Sprintf(`ALTER DATABASE "%s" ALLOW_CONNECTIONS false`, database.name),
		)
		require.NoError(t, err)

		cancelRestore()
		archiveErr := <-restoreErrors
		require.ErrorContains(t, archiveErr, "pg_restore failed")
		assert.NotContains(t, archiveErr.Error(), "timescaledb_post_restore failed")
		assert.Contains(t, restoreLog.String(), "target database may be left in restoring mode")

		_, err = serverConnection.Exec(
			t.Context(),
			fmt.Sprintf(`ALTER DATABASE "%s" ALLOW_CONNECTIONS true`, database.name),
		)
		require.NoError(t, err)
		assert.True(t, database.isTimescaleRestoring(t))
	})

	t.Run("fails after a successful archive restore when cleanup cannot connect", func(t *testing.T) {
		source := server.createDatabase(t)
		source.exec(t, "CREATE TABLE cleanup_rows(id integer)", "INSERT INTO cleanup_rows VALUES (1)")
		spec := newTimescaleRestoreSpec(t)
		dumpSpec := newStubDumpSpec(spec.ClientTools, spec.Version)
		dumpSpec.IncludeSchemas = []string{"public"}

		var archive bytes.Buffer
		require.NoError(t, source.newAdminSource().Dump(t.Context(), dumpSpec, &archive))

		target := server.createDatabase(t)
		serverConnection := server.connectAs(t, getAdminRole(), containers.PostgresDatabase)
		spec.NewCleanupContext = func(ctx context.Context) (context.Context, context.CancelFunc) {
			_, err := serverConnection.Exec(
				ctx,
				fmt.Sprintf(`ALTER DATABASE "%s" ALLOW_CONNECTIONS false`, target.name),
			)
			require.NoError(t, err)
			return context.WithCancel(context.WithoutCancel(ctx))
		}

		err := target.newAdminSource().Restore(t.Context(), spec, &archive)
		require.ErrorContains(t, err, "timescaledb_post_restore failed")

		var restoredRows int
		require.NoError(t, target.admin.QueryRow(t.Context(), "SELECT count(*) FROM cleanup_rows").Scan(&restoredRows))
		assert.Equal(t, 1, restoredRows)

		_, err = serverConnection.Exec(
			t.Context(),
			fmt.Sprintf(`ALTER DATABASE "%s" ALLOW_CONNECTIONS true`, target.name),
		)
		require.NoError(t, err)
		assert.True(t, target.isTimescaleRestoring(t))
	})
}

func newTimescaleRestoreSpec(t *testing.T) RestoreSpec {
	t.Helper()
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	clientToolsDir, err := clienttools.FindBundleRoot(workingDirectory)
	require.NoError(t, err)

	spec := newStubRestoreSpec(t, pgshared.ClientTools{Root: clientToolsDir})
	spec.Version = pgshared.Version17
	spec.IsTimescaleDB = true
	spec.Logger = newTestLogger(t)
	return spec
}

func (d testDatabase) isTimescaleRestoring(t *testing.T) bool {
	t.Helper()
	connection := d.connectAs(t, getAdminRole())
	var restoringMode string
	require.NoError(t, connection.QueryRow(t.Context(), "SHOW timescaledb.restoring").Scan(&restoringMode))
	require.NoError(t, connection.Close(t.Context()))
	return restoringMode == "on"
}
