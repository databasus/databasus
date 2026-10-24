package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

// plantHistoryFileOnSource writes a synthetic .history file into the source
// cluster's pg_wal so UploadHistoryFile's pg_read_binary_file read succeeds —
// timeline 1 never produces a .history file, so without this the upload would
// fail at the read step before reaching the catalog insert under test. The
// shared source PG connects as a superuser, so a server-side COPY ... TO can
// create the file. timelineID is intentionally far above any real promotion
// timeline on the shared container, and there is no SQL to remove a server
// file, but a .history for a nonexistent timeline is inert.
func plantHistoryFileOnSource(t *testing.T, conn *pgx.Conn, timelineID int) {
	t.Helper()

	var dataDir string
	require.NoError(t, conn.QueryRow(context.Background(), "SHOW data_directory").Scan(&dataDir))

	historyPath := fmt.Sprintf("%s/pg_wal/%s", dataDir, walmath.FormatHistoryFilename(uint32(timelineID)))

	_, err := conn.Exec(context.Background(),
		fmt.Sprintf("COPY (SELECT '1'::text) TO '%s'", historyPath))
	require.NoError(t, err)
}

// Test_UploadHistoryFile_KeysRowOnParentDatabaseID asserts the WAL history-file
// catalog row is keyed on the parent databases.id, not the
// postgresql_physical_databases PK. fk_physical_wal_history_files_database_id
// REFERENCES databases(id), so keying on the physical PK fails the insert with
// SQLSTATE 23503 and aborts the WAL-stream supervisor before any segment is
// recorded (issue #643). The two ids are distinct uuids, so the wrong one is
// rejected outright.
func Test_UploadHistoryFile_KeysRowOnParentDatabaseID(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	sourceDB := fixture.DB.PostgresqlPhysical
	require.NotEqual(t, fixture.DB.ID, sourceDB.ID,
		"fixture must expose a physical PK distinct from the parent databases.id to exercise the FK")

	const timelineID = 1000

	adminConn := OpenAdminConn(t, fixture)
	plantHistoryFileOnSource(t, adminConn, timelineID)

	conn, err := sourceDB.OpenInspectionConn(t.Context(), encryption.GetFieldEncryptor())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	historyRepo := physical_repositories.GetWalHistoryRepository()

	row, err := UploadHistoryFile(t.Context(), HistoryUploadSpec{
		Conn:           conn,
		TimelineID:     timelineID,
		FileStore:      newMockWalStoreFor(newMockWalStorage()),
		SourceDB:       sourceDB,
		StorageID:      fixture.Storage.ID,
		HistoryRepo:    historyRepo,
		Encryption:     backups_core_enums.BackupEncryptionNone,
		FieldEncryptor: encryption.GetFieldEncryptor(),
		Logger:         logger.GetLogger(),
	})
	require.NoError(t, err, "history insert must not violate fk_physical_wal_history_files_database_id")
	require.NotNil(t, row)
	t.Cleanup(func() { _ = historyRepo.DeleteByID(row.ID) })

	assert.Equal(t, fixture.DB.ID, row.DatabaseID,
		"history row must be keyed on the parent databases.id, not the physical PK")

	byParent, err := historyRepo.FindByDatabaseTimeline(fixture.DB.ID, timelineID)
	require.NoError(t, err)
	require.NotNil(t, byParent, "history row must be findable by the parent databases.id (chain assembly reads by it)")

	byPhysicalPK, err := historyRepo.FindByDatabaseTimeline(sourceDB.ID, timelineID)
	require.NoError(t, err)
	assert.Nil(t, byPhysicalPK, "history row must not be keyed on the physical PK")
}
