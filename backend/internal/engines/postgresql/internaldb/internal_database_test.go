package internaldb_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/engines/postgresql/internaldb"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/testdb"
)

func TestInternalDatabase(t *testing.T) {
	t.Parallel()

	t.Run("restores the snapshot with its migration history into a staging database", func(t *testing.T) {
		t.Parallel()
		sourceDsn := testdb.Open(t)
		sourceDb := openPgx(t, sourceDsn)
		_, err := sourceDb.Exec(t.Context(),
			`INSERT INTO workspaces (id, name, created_at) VALUES ($1, 'Snapshot workspace', now())`, uuid.New())
		require.NoError(t, err)
		sourceConnection, err := internaldb.ParseConnection(sourceDsn)
		require.NoError(t, err)

		snapshotTransaction, err := sourceDb.BeginTx(t.Context(), pgx.TxOptions{
			IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly, DeferrableMode: "", BeginQuery: "", CommitQuery: "",
		})
		require.NoError(t, err)
		var snapshotID string
		require.NoError(t, snapshotTransaction.QueryRow(t.Context(), "SELECT pg_export_snapshot()").Scan(&snapshotID))
		_, err = openPgx(t, sourceDsn).Exec(t.Context(), `UPDATE workspaces SET name = 'Changed after the snapshot'`)
		require.NoError(t, err)

		var dump bytes.Buffer
		err = internaldb.Dump(t.Context(), internaldb.DumpSpec{
			ClientTools: getClientTools(t),
			Connection:  sourceConnection,
			SnapshotID:  snapshotID,
			Logger:      slog.New(slog.DiscardHandler),
		}, &dump)
		require.NoError(t, err)
		require.NoError(t, snapshotTransaction.Rollback(t.Context()))

		stagingConnection := createStagingDatabase(t, sourceConnection)
		err = internaldb.Restore(t.Context(), internaldb.RestoreSpec{
			ClientTools: getClientTools(t),
			Connection:  stagingConnection,
			Logger:      slog.New(slog.DiscardHandler),
		}, &dump)
		require.NoError(t, err)

		stagingDb := openPgx(t, connectionDsn(stagingConnection))
		var workspaceName string
		require.NoError(t, stagingDb.QueryRow(t.Context(), "SELECT name FROM workspaces").Scan(&workspaceName))
		sourceHistoryCount := countMigrationHistory(t, openPgx(t, sourceDsn))
		stagingHistoryCount := countMigrationHistory(t, stagingDb)

		assert.Equal(t, "Snapshot workspace", workspaceName)
		assert.Positive(t, sourceHistoryCount)
		assert.Equal(t, sourceHistoryCount, stagingHistoryCount)
	})

	t.Run("refuses a dump without an exported snapshot", func(t *testing.T) {
		t.Parallel()

		err := internaldb.Dump(t.Context(), internaldb.DumpSpec{
			ClientTools: getClientTools(t),
			Connection:  internaldb.Connection{Host: "", Port: 0, User: "", Password: "", Database: ""},
			SnapshotID:  "",
			Logger:      slog.New(slog.DiscardHandler),
		}, &bytes.Buffer{})

		require.ErrorIs(t, err, internaldb.ErrSnapshotIDRequired)
	})

	t.Run("fails and leaves the staging database empty when the archive is not a dump", func(t *testing.T) {
		t.Parallel()
		sourceConnection, err := internaldb.ParseConnection(testdb.OpenEmpty(t))
		require.NoError(t, err)
		stagingConnection := createStagingDatabase(t, sourceConnection)

		err = internaldb.Restore(t.Context(), internaldb.RestoreSpec{
			ClientTools: getClientTools(t),
			Connection:  stagingConnection,
			Logger:      slog.New(slog.DiscardHandler),
		}, strings.NewReader("not a custom-format archive"))

		require.ErrorIs(t, err, internaldb.ErrRestoreFailed)
		var tableCount int64
		require.NoError(t, openPgx(t, connectionDsn(stagingConnection)).QueryRow(t.Context(),
			"SELECT count(*) FROM pg_tables WHERE schemaname = 'public'").Scan(&tableCount))
		assert.Zero(t, tableCount)
	})

	t.Run("refuses to create a database outside the staging prefix", func(t *testing.T) {
		t.Parallel()

		err := internaldb.CreateStagingDatabase(t.Context(), internaldb.Connection{
			Host: "", Port: 0, User: "", Password: "", Database: "",
		}, "databasus")

		require.ErrorIs(t, err, internaldb.ErrStagingNameInvalid)
	})
}

func createStagingDatabase(t *testing.T, server internaldb.Connection) internaldb.Connection {
	t.Helper()

	name := internaldb.StagingPrefix + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	require.NoError(t, internaldb.CreateStagingDatabase(t.Context(), server, name))
	t.Cleanup(func() {
		_ = internaldb.DropStagingDatabase(context.WithoutCancel(t.Context()), server, name)
	})

	return server.WithDatabase(name)
}

func openPgx(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()

	connection, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })

	return connection
}

func connectionDsn(connection internaldb.Connection) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		connection.Host, connection.Port, connection.User, connection.Password, connection.Database)
}

func countMigrationHistory(t *testing.T, connection *pgx.Conn) int64 {
	t.Helper()

	var count int64
	require.NoError(t, connection.QueryRow(t.Context(), "SELECT count(*) FROM goose_db_version").Scan(&count))

	return count
}

func getClientTools(t *testing.T) pgshared.ClientTools {
	t.Helper()

	workingDir, err := os.Getwd()
	require.NoError(t, err)
	root, err := clienttools.FindBundleRoot(workingDir)
	require.NoError(t, err)

	return pgshared.ClientTools{Root: root}
}
