package physical

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/process"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // Versions and ordering cases use one source server sequentially.
func TestWalTransactionTimeOrder(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			server := startTestServer(t, image, containers.WithMaxPreparedTransactions(4))

			for _, scenario := range []struct {
				name              string
				shouldUsePrepared bool
			}{
				{name: "preserves the maximum time when a later commit has an older timestamp"},
				{name: "preserves trusted time when a verified prepared commit has an older timestamp", shouldUsePrepared: true},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					inspection := server.openInspection(t, getAdminRole())
					table := `"wal_order_` + uuid.NewString() + `"`
					execStatements(t, inspection.conn, "CREATE TABLE "+table+" (id integer)")

					execStatements(t, inspection.conn, "SELECT pg_switch_wal()")

					var filename string
					require.NoError(t, inspection.conn.QueryRow(t.Context(),
						"SELECT pg_walfile_name(pg_current_wal_insert_lsn())").Scan(&filename))

					execStatements(t, inspection.conn, "INSERT INTO "+table+" VALUES (1)")

					if scenario.shouldUsePrepared {
						identifier := "'wal-order-" + uuid.NewString() + "'"
						execStatements(t, inspection.conn, "BEGIN", "INSERT INTO "+table+" VALUES (2)",
							"PREPARE TRANSACTION "+identifier, "COMMIT PREPARED "+identifier)
					} else {
						execStatements(t, inspection.conn, "INSERT INTO "+table+" VALUES (2)")
					}

					var endLSN walmath.LSN
					var observedAt time.Time
					require.NoError(
						t,
						inspection.conn.QueryRow(t.Context(), "SELECT pg_current_wal_insert_lsn(), clock_timestamp()").
							Scan(&endLSN, &observedAt),
					)

					execStatements(t, inspection.conn, "SELECT pg_switch_wal()")

					var body []byte
					require.NoError(t, inspection.conn.QueryRow(t.Context(),
						"SELECT pg_read_binary_file('pg_wal/' || $1)", filename).Scan(&body))

					path := filepath.Join(t.TempDir(), filename)
					require.NoError(t, os.WriteFile(path, body, 0o600))

					spec := WalTransactionTimeSpec{
						ClientTools: getWalTimeClientTools(t), Version: server.version, LocalPath: path, EndLSN: endLSN,
					}
					maximum := observedAt.UTC().Add(time.Microsecond).Truncate(time.Microsecond)
					setEarlierWalCommitMaximum(t, spec, maximum)

					transactionAt, err := GetWalTransactionTime(t.Context(), spec)

					require.NoError(t, err)
					require.NotNil(t, transactionAt)
					require.Equal(t, maximum, transactionAt.UTC())
				})
			}
		})
	}
}

func setEarlierWalCommitMaximum(t *testing.T, spec WalTransactionTimeSpec, maximum time.Time) {
	t.Helper()

	var decoded bytes.Buffer
	var diagnostics bytes.Buffer
	err := process.Run(t.Context(), process.Spec{
		Path: spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgWaldump),
		Args: []string{"--rmgr=Transaction", "--end=" + spec.EndLSN.String(), filepath.Base(spec.LocalPath)},
		Dir:  filepath.Dir(spec.LocalPath), Env: []string{"LC_ALL=C", "TZ=UTC"},
		Stdout: &decoded, Stderr: &diagnostics,
	})
	require.NoError(t, err, "%s", diagnostics.String())

	var commitLine string
	for line := range strings.SplitSeq(decoded.String(), "\n") {
		if strings.Contains(line, ", desc: COMMIT ") {
			commitLine = line
			break
		}
	}

	require.NotEmpty(t, commitLine)

	header, _, _ := strings.Cut(commitLine, ", desc: COMMIT ")
	_, location, _ := strings.Cut(header, ", lsn: ")
	location, _, _ = strings.Cut(location, ", prev ")
	recordLSN, err := walmath.ParseLSN(location)
	require.NoError(t, err)

	body, err := os.ReadFile(spec.LocalPath)
	require.NoError(t, err)

	position := int(recordLSN % walmath.LSN(walmath.DefaultSegmentSizeBytes))
	length := int(binary.LittleEndian.Uint32(body[position:]))
	require.GreaterOrEqual(t, length, 34)
	require.LessOrEqual(t, length, 281, "the isolated INSERT must use a short main-data header")

	positions := make([]int, length)
	record := make([]byte, length)
	for index := range positions {
		if position%8192 == 0 {
			position += 24
		}

		positions[index] = position
		record[index] = body[position]
		position++
	}

	require.Equal(t, []byte{255, byte(length - 26)}, record[24:26])

	checksumTable := crc32.MakeTable(crc32.Castagnoli)
	checksum := crc32.Update(crc32.Checksum(record[24:], checksumTable), checksumTable, record[:20])
	require.Equal(t, checksum, binary.LittleEndian.Uint32(record[20:24]))

	postgresEpoch := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	binary.LittleEndian.PutUint64(record[26:], uint64(maximum.Sub(postgresEpoch).Microseconds()))
	checksum = crc32.Update(crc32.Checksum(record[24:], checksumTable), checksumTable, record[:20])
	binary.LittleEndian.PutUint32(record[20:24], checksum)

	for index, offset := range positions {
		body[offset] = record[index]
	}

	require.NoError(t, os.WriteFile(spec.LocalPath, body, 0o600))
}
