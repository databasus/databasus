package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type backupRequests struct {
	IsFullRequested        bool
	IsIncrementalRequested bool
}

func (catalog *catalogFixture) getBackupRequests(t *testing.T) backupRequests {
	t.Helper()

	var requests []backupRequests

	catalog.App.QuerySQL(t, &requests,
		`SELECT force_full_requested_at IS NOT NULL AS is_full_requested,
                force_incremental_requested_at IS NOT NULL AS is_incremental_requested
         FROM backups_configs_physical WHERE database_id=?`, catalog.DatabaseID)
	require.Len(t, requests, 1)

	return requests[0]
}
