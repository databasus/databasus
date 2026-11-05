package mariadb

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
	t.Run("negotiates the compress flag with a stock MariaDB", func(t *testing.T) {
		endpoint := containers.StartMariadb(t, "mariadb:10.11")
		var dumpLog bytes.Buffer
		source := NewSource(Settings{
			Host:      endpoint.Host,
			Port:      endpoint.Port,
			Username:  containers.MariadbUsername,
			Password:  containers.MariadbPassword,
			Database:  containers.MariadbDatabase,
			IsHTTPS:   false,
			SSHTunnel: nil,
		}, getClientTools(t))

		err := source.Dump(t.Context(), DumpSpec{
			Version:         Version1011,
			ExcludeTables:   nil,
			Privileges:      nil,
			IsExcludeEvents: false,
			Logger:          slog.New(slog.NewTextHandler(&dumpLog, nil)),
		}, io.Discard)

		require.NoError(t, err)
		assert.Contains(t, dumpLog.String(), "compression=--compress")
	})
}
