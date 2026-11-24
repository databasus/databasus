package postgresql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// TestBackupPostgresql18 pins issue #921: PostgreSQL 18.7 stores the internal column option
// pg_attver, an older pg_dump writes it into the dump, and a server before 18.7 rejects the dump on
// restore.
func TestBackupPostgresql18(t *testing.T) {
	t.Parallel()

	t.Run("writes no pg_attver column option when the server is on the pinned minor", func(t *testing.T) {
		t.Parallel()

		endpoint := containers.StartPostgres(t, "postgres:18")
		server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
		server.createFilledTable(t)

		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, server.newSettings())
		backup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)

		dump := tm.DownloadBackup(t, backup.Id)

		assert.NotEmpty(t, dump)
		assert.False(t, bytes.Contains(dump, []byte("pg_attver")), "the dump carries the pg_attver column option")
	})
}
