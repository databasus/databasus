package backuping_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func holdStorageWrites(t *testing.T, team *backupTeam) func() {
	t.Helper()
	team.App.InstallStorageWriteBarrier(t)
	storageLock := team.App.BeginSQLTransaction(t)

	var storageIDs []string
	storageLock.QuerySQL(t, &storageIDs, "SELECT id FROM storages WHERE id=? FOR NO KEY UPDATE", team.StorageID)
	require.Len(t, storageIDs, 1)

	return func() { storageLock.Rollback(t) }
}
