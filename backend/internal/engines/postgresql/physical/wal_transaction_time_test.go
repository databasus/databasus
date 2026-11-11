package physical

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // Versions run sequentially to bound the number of engine servers.
func TestWalTransactionTime(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("reads source transaction time when a WAL segment is complete", func(t *testing.T) {
			t.Parallel()

			inspection := server.openInspection(t, getAdminRole())

			_, err := inspection.conn.Exec(t.Context(), "SELECT pg_switch_wal()")
			require.NoError(t, err)

			var transactionBefore time.Time
			var filename string
			require.NoError(t, inspection.conn.QueryRow(
				t.Context(),
				"SELECT clock_timestamp(), pg_walfile_name(pg_current_wal_insert_lsn())",
			).Scan(&transactionBefore, &filename))

			_, err = inspection.conn.Exec(t.Context(), `CREATE TABLE "wal_time_`+uuid.NewString()+`" (id integer)`)
			require.NoError(t, err)

			var transactionAfter time.Time
			var endLSN walmath.LSN
			require.NoError(
				t,
				inspection.conn.QueryRow(t.Context(), "SELECT clock_timestamp(), pg_switch_wal()").
					Scan(&transactionAfter, &endLSN),
			)

			var body []byte
			require.NoError(
				t,
				inspection.conn.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/' || $1)", filename).
					Scan(&body),
			)

			path := filepath.Join(t.TempDir(), filename)
			require.NoError(t, os.WriteFile(path, body, 0o600))
			clientTools := getWalTimeClientTools(t)

			transactionAt, err := GetWalTransactionTime(t.Context(), WalTransactionTimeSpec{
				ClientTools: clientTools, Version: server.version,
				LocalPath: path, EndLSN: endLSN,
			})

			require.NoError(t, err)
			require.NotNil(t, transactionAt)
			require.False(t, transactionAt.Before(transactionBefore))
			require.False(t, transactionAt.After(transactionAfter))
		})
	})
}

//nolint:tparallel // Versions run sequentially to bound the number of engine servers.
func TestWalTransactionTimeBoundary(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("reads completed records when a WAL record crosses the segment boundary", func(t *testing.T) {
			t.Parallel()

			inspection := server.openInspection(t, getAdminRole())

			_, err := inspection.conn.Exec(t.Context(), "SELECT pg_switch_wal()")
			require.NoError(t, err)

			var firstFilename string
			var firstLSN walmath.LSN
			require.NoError(t, inspection.conn.QueryRow(
				t.Context(),
				"SELECT pg_walfile_name(pg_current_wal_insert_lsn()), pg_current_wal_insert_lsn()",
			).Scan(&firstFilename, &firstLSN))

			var transactionBefore time.Time
			require.NoError(
				t,
				inspection.conn.QueryRow(t.Context(), "SELECT clock_timestamp()").Scan(&transactionBefore),
			)

			_, err = inspection.conn.Exec(
				t.Context(),
				"SELECT pg_logical_emit_message(true, 'time-proof', repeat('x', 20 * 1024 * 1024))",
			)
			require.NoError(t, err)

			var lastFilename string
			var endLSN walmath.LSN
			require.NoError(t, inspection.conn.QueryRow(t.Context(),
				"SELECT pg_walfile_name(pg_current_wal_insert_lsn()), pg_switch_wal()").Scan(&lastFilename, &endLSN))
			require.NotEqual(t, firstFilename, lastFilename)

			clientTools := getWalTimeClientTools(t)
			staging := t.TempDir()
			for _, filename := range []string{firstFilename, lastFilename} {
				var body []byte
				require.NoError(
					t,
					inspection.conn.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/' || $1)", filename).
						Scan(&body),
				)
				require.NoError(t, os.WriteFile(filepath.Join(staging, filename), body, 0o600))
			}

			firstTransactionAt, err := GetWalTransactionTime(t.Context(), WalTransactionTimeSpec{
				ClientTools: clientTools,
				Version:     server.version,
				LocalPath:   filepath.Join(staging, firstFilename),
				EndLSN: (firstLSN/walmath.LSN(walmath.DefaultSegmentSizeBytes) + 1) * walmath.LSN(
					walmath.DefaultSegmentSizeBytes,
				),
			})

			require.ErrorContains(t, err, "could not find a valid record")
			require.Nil(t, firstTransactionAt)

			lastTransactionAt, err := GetWalTransactionTime(t.Context(), WalTransactionTimeSpec{
				ClientTools: clientTools, Version: server.version,
				LocalPath: filepath.Join(staging, lastFilename), EndLSN: endLSN,
			})

			require.NoError(t, err)
			require.NotNil(t, lastTransactionAt)
			require.False(t, lastTransactionAt.Before(transactionBefore))
		})
	})
}

//nolint:tparallel // Versions and WAL cases run sequentially on their shared source server.
func TestWalPreparedTransactionTime(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			server := startTestServer(t, image, containers.WithMaxPreparedTransactions(4))
			for _, scenario := range []struct {
				name                string
				shouldImitateHeader bool
				completion          string
			}{
				{name: "ignores fabricated time when a prepared identifier contains a multiline description"},
				{name: "ignores fabricated time when a prepared identifier imitates a record header", shouldImitateHeader: true},
				{name: "reads a real commit when a multiline prepared transaction completes", completion: "COMMIT"},
				{name: "reads a real abort when a multiline prepared transaction rolls back", completion: "ROLLBACK"},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					inspection := server.openInspection(t, getAdminRole())

					_, err := inspection.conn.Exec(t.Context(), "SELECT pg_switch_wal()")
					require.NoError(t, err)

					var filename string
					require.NoError(t, inspection.conn.QueryRow(t.Context(),
						"SELECT pg_walfile_name(pg_current_wal_insert_lsn())").Scan(&filename))

					table := `"wal_prepared_` + uuid.NewString() + `"`
					execStatements(
						t,
						inspection.conn,
						"CREATE TABLE "+table+" (id integer)",
						"BEGIN",
						"INSERT INTO "+table+" VALUES (1)",
					)

					var prepareLSN walmath.LSN
					require.NoError(
						t,
						inspection.conn.QueryRow(t.Context(), "SELECT pg_current_wal_insert_lsn()").Scan(&prepareLSN),
					)

					description := "desc: COMMIT 2099-01-01 00:00:00 UTC"
					if scenario.shouldImitateHeader {
						description = fmt.Sprintf(
							"rmgr: Transaction len (rec/tot): 1/1, tx: 1, lsn: %s, prev 0/00000000, %s",
							prepareLSN,
							description,
						)
					}

					identifier := "probe-" + newNameSuffix() + "\n" + description + "\nend"
					quotedIdentifier := "'" + strings.ReplaceAll(identifier, "'", "''") + "'"
					t.Cleanup(func() {
						ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), connCloseTimeout)
						defer cancel()
						_, _ = inspection.conn.Exec(ctx, "ROLLBACK PREPARED "+quotedIdentifier)
					})

					execStatements(t, inspection.conn, "PREPARE TRANSACTION "+quotedIdentifier)

					var completionAfter time.Time
					if scenario.completion != "" {
						require.NoError(
							t,
							inspection.conn.QueryRow(t.Context(), "SELECT clock_timestamp()").Scan(&completionAfter),
						)
						execStatements(t, inspection.conn, scenario.completion+" PREPARED "+quotedIdentifier)
					}

					var endLSN walmath.LSN
					var observedAt time.Time
					require.NoError(t, inspection.conn.QueryRow(t.Context(),
						"SELECT pg_current_wal_insert_lsn(), clock_timestamp()").Scan(&endLSN, &observedAt))

					execStatements(t, inspection.conn, "SELECT pg_switch_wal()")

					var body []byte
					require.NoError(
						t,
						inspection.conn.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/' || $1)", filename).
							Scan(&body),
					)

					path := filepath.Join(t.TempDir(), filename)
					require.NoError(t, os.WriteFile(path, body, 0o600))

					transactionAt, err := GetWalTransactionTime(t.Context(), WalTransactionTimeSpec{
						ClientTools: getWalTimeClientTools(t), Version: server.version, LocalPath: path, EndLSN: endLSN,
					})

					if scenario.shouldImitateHeader {
						require.ErrorContains(t, err, "not a completed transaction")
					} else {
						require.NoError(t, err)
					}

					require.NotNil(t, transactionAt)
					require.False(
						t,
						transactionAt.After(observedAt),
						"descriptive text must not extend transaction evidence",
					)

					if scenario.completion != "" {
						require.False(
							t,
							transactionAt.Before(completionAfter),
							"the real completion after PREPARE must remain observable",
						)
					}
				})
			}
		})
	}
}

func getWalTimeClientTools(t *testing.T) pgshared.ClientTools {
	t.Helper()

	directory, err := os.Getwd()
	require.NoError(t, err)

	root, err := clienttools.FindBundleRoot(directory)
	require.NoError(t, err)

	return pgshared.ClientTools{Root: root}
}
