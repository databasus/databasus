package backuping_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func holdStorageWrites(t *testing.T, team *backupTeam) func() {
	t.Helper()
	installStorageWriteBarrier(t, team.App)
	storageLock := team.App.BeginSQLTransaction(t)

	var storageIDs []string
	storageLock.QuerySQL(t, &storageIDs, "SELECT id FROM storages WHERE id=? FOR NO KEY UPDATE", team.StorageID)
	require.Len(t, storageIDs, 1)

	return func() { storageLock.Rollback(t) }
}

func installStorageWriteBarrier(t *testing.T, app *apptest.App) {
	t.Helper()
	app.ExecuteSQL(t, `CREATE FUNCTION hold_physical_stream_write() RETURNS trigger AS $$ BEGIN
		IF NEW.not_before>NEW.created_at THEN
		PERFORM id FROM storages WHERE id=NEW.storage_id FOR SHARE;
		END IF; RETURN NEW; END $$ LANGUAGE plpgsql`)
	app.ExecuteSQL(
		t,
		`CREATE TRIGGER hold_physical_stream_write BEFORE INSERT ON storages_pending_deletions FOR EACH ROW EXECUTE FUNCTION hold_physical_stream_write()`,
	)
}
