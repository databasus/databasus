package database_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	t.Run("reads a stored time back in UTC whatever the process time zone", func(t *testing.T) {
		t.Parallel()

		db, err := database.Open(t.Context(), testdb.OpenEmpty(t))
		require.NoError(t, err)
		t.Cleanup(func() { closePool(db) })

		storedTime := time.Date(2026, time.November, 25, 4, 0, 0, 0, time.FixedZone("UTC-3", -3*60*60))

		var readBackTime time.Time
		err = db.Raw(`SELECT ?::timestamptz`, storedTime).Row().Scan(&readBackTime)

		require.NoError(t, err)
		assert.Equal(t, time.UTC, readBackTime.Location())
		assert.True(t, storedTime.Equal(readBackTime))
	})
}
