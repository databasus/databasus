package report

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/verification"
)

func TestDatabaseEntry(t *testing.T) {
	t.Parallel()
	t.Run("keeps engine facts when mapping each supported database type", func(t *testing.T) {
		t.Parallel()
		for _, databaseType := range []databases.Type{databases.TypePostgresqlLogical, databases.TypePostgresqlPhysical, databases.TypeMysql, databases.TypeMariadb, databases.TypeMongodb} {
			database := databases.TelemetryEntry{
				Type:               databaseType,
				Version:            "server-version",
				IsSSHTunnelEnabled: true,
			}

			entry := buildDatabaseEntry(database)

			assert.Equal(t, string(databaseType), entry.Type)
			assert.Equal(t, "server-version", entry.Version)
			assert.True(t, entry.IsSSHTunnelEnabled)
		}
	})
	t.Run("omits unused fields when a verification runs after a backup", func(t *testing.T) {
		t.Parallel()
		config := verification.VerificationConfig{ScheduleType: verification.ScheduleAfterBackup}
		encoded, err := json.Marshal(buildVerificationEntry(config))
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "intervalType")

		encoded, err = json.Marshal(DatabaseEntry{Type: "MYSQL", Version: "8.4"})
		require.NoError(t, err)
		for _, field := range []string{"backupType", "backupSizeMb", "rawSizeMb", "verification"} {
			assert.NotContains(t, string(encoded), field)
		}
	})
}
