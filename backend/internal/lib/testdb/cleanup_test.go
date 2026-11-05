package testdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveAbandonedDatabases(t *testing.T) {
	t.Parallel()

	t.Run("drops only old test databases without connections", func(t *testing.T) {
		t.Parallel()
		abandonedName := createNamedDatabase(t, newTestDatabaseNameAt(t, time.Now()))
		connectedName := createNamedDatabase(t, newTestDatabaseNameAt(t, time.Now()))
		openDatabase(t, replaceDatabaseName(readTestDsn(t), connectedName))
		freshName := createNamedDatabase(t, newTestDatabaseNameAt(t, time.Now().Add(cleanupClockShift)))
		templateName := createNamedDatabase(t, templateDatabasePrefix+randomHex(t))
		foreignName := createNamedDatabase(t, "databasus_cleanup_probe_"+randomHex(t))
		scope := cleanupScope{
			now:           time.Now().Add(cleanupClockShift),
			databaseNames: []string{abandonedName, connectedName, freshName, templateName, foreignName},
		}

		require.NoError(t, removeAbandonedDatabases(t.Context(), openSystemDatabase(t), scope))

		assert.False(t, isDatabasePresent(t, abandonedName), "old database without connections")
		assert.True(t, isDatabasePresent(t, connectedName), "old database with a connection")
		assert.True(t, isDatabasePresent(t, freshName), "fresh database")
		assert.True(t, isDatabasePresent(t, templateName), "template")
		assert.True(t, isDatabasePresent(t, foreignName), "database outside the name pattern")
	})
}
