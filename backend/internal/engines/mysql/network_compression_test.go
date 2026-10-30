package mysql

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestDumpNetworkCompression(t *testing.T) {
	t.Run("negotiates zstd with a stock MySQL 8.0", func(t *testing.T) {
		endpoint := containers.StartMysql(t, "mysql:8.0")

		dumpLog := dumpWithRealClient(t, endpoint)

		assert.Contains(t, dumpLog, `compression="--compression-algorithms=zstd --zstd-compression-level=5"`)
	})

	t.Run("falls back to uncompressed when the server refuses compression", func(t *testing.T) {
		endpoint := containers.StartMysqlWithoutCompression(t, "mysql:8.0")

		dumpLog := dumpWithRealClient(t, endpoint)

		assert.Contains(t, dumpLog, "compression=uncompressed")
	})
}

// dumpWithRealClient dumps the container's database with the shipped mysqldump and returns the log
// of the dump, which names the negotiated compression.
func dumpWithRealClient(t *testing.T, endpoint containers.Endpoint) string {
	t.Helper()

	var dumpLog bytes.Buffer
	source := NewSource(Settings{
		Host:      endpoint.Host,
		Port:      endpoint.Port,
		Username:  containers.MysqlUsername,
		Password:  containers.MysqlPassword,
		Database:  containers.MysqlDatabase,
		IsHTTPS:   false,
		SSHTunnel: nil,
	}, getClientTools(t))

	err := source.Dump(t.Context(), DumpSpec{
		Version:       Version80,
		ExcludeTables: nil,
		Privileges:    nil,
		Logger:        slog.New(slog.NewTextHandler(&dumpLog, nil)),
	}, io.Discard)
	require.NoError(t, err)

	return dumpLog.String()
}
