package apptest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLTransaction(t *testing.T) {
	t.Parallel()

	t.Run("commits a held transaction when the test releases it", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		transaction := testApp.BeginSQLTransaction(t)
		transaction.ExecuteSQL(
			t,
			"INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, true)",
			isolationMarkerVersion,
		)
		var markerCount int
		testApp.QuerySQL(
			t,
			&markerCount,
			"SELECT count(*) FROM goose_db_version WHERE version_id = ?",
			isolationMarkerVersion,
		)
		require.Zero(t, markerCount)

		transaction.Commit(t)

		testApp.QuerySQL(
			t,
			&markerCount,
			"SELECT count(*) FROM goose_db_version WHERE version_id = ?",
			isolationMarkerVersion,
		)
		assert.Equal(t, 1, markerCount)
	})
}
