package databases_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	backups_bridge "databasus-backend/internal/features/backups/config"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/util/testing/containers"
)

const storageFileDeletionJob = "storage_file_deletion"

func TestDeleteDatabase(t *testing.T) {
	postgresServer := containers.StartPostgres(t, "postgres:16")

	t.Run("deletes the database with its backups and removes their files", func(t *testing.T) {
		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		storedBackup := insertPublishedLogicalBackup(t, tm, logicalBackupSpec{
			databaseID: savedDatabase.Id,
			storageID:  storageID,
			status:     "COMPLETED",
		})

		response := deleteDatabase(t, tm.owner, savedDatabase.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, tm.listDatabaseIDs(t))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM backups_logical WHERE database_id = ?", savedDatabase.Id))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM backups_configs WHERE database_id = ?",
			savedDatabase.Id))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM databases_postgresql_logical WHERE database_id = ?",
			savedDatabase.Id))

		tm.app.RunJobPass(t, storageFileDeletionJob)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, storedBackup.fileName))
	})

	t.Run("returns 409 in_use and deletes nothing while a logical backup is in progress", func(t *testing.T) {
		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		runningBackup := insertPublishedLogicalBackup(t, tm, logicalBackupSpec{
			databaseID: savedDatabase.Id,
			storageID:  storageID,
			status:     "IN_PROGRESS",
		})

		response := deleteDatabase(t, tm.owner, savedDatabase.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
		assert.Equal(t, 1, tm.countRows(t, "SELECT COUNT(*) FROM backups_logical WHERE id = ?", runningBackup.id))
	})

	t.Run("keeps the database, its backups and their files when deleting the backup records fails", func(t *testing.T) {
		tm := newTeam(t)
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		storedBackup := insertPublishedLogicalBackup(t, tm, logicalBackupSpec{
			databaseID: savedDatabase.Id,
			storageID:  storageID,
			status:     "COMPLETED",
		})
		refuseLogicalBackupDeletion(t, tm.app)

		response := deleteDatabase(t, tm.owner, savedDatabase.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusInternalServerError,
			apiclient.ProblemCodeInternal)
		assert.NotContains(t, string(response.Body), refusedBackupDeletionMessage)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
		assert.Equal(t, 1, tm.countRows(t, "SELECT COUNT(*) FROM backups_logical WHERE id = ?", storedBackup.id))
		assert.Zero(t, tm.countRows(t, "SELECT COUNT(*) FROM storages_pending_deletions WHERE storage_id = ?",
			storageID))

		tm.app.RunJobPass(t, storageFileDeletionJob)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, storedBackup.fileName))
	})
}

func TestDeletePhysicalDatabase(t *testing.T) {
	physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")

	t.Run("stops the WAL streamer and drops its replication slots after the database is gone", func(t *testing.T) {
		tm := newTeam(t)
		savedDatabase := tm.createDatabase(t,
			newPostgresqlPhysicalSettings(t, physicalServer, apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM))
		slotNames := readReplicationSlotNames(t, tm, savedDatabase.Id)
		walSlotName := slotNames[0]
		adminConn := connectAsAdmin(t, physicalServer, containers.PostgresDatabase)

		for _, slotName := range slotNames {
			_, err := adminConn.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1, true)", slotName)
			require.NoError(t, err)
		}

		walStreamer := startWalStreamerTask(t, physicalServer, savedDatabase.Id, walSlotName)

		response := deleteDatabase(t, tm.owner, savedDatabase.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, tm.listDatabaseIDs(t))

		testwait.Until(t, "the WAL streamer task is cancelled", func(context.Context) error {
			select {
			case <-walStreamer.stopped:
				return nil
			default:
				return errors.New("the WAL streamer task is still running")
			}
		})

		testwait.Until(t, "the replication slots are dropped", func(ctx context.Context) error {
			var remainingSlotCount int
			err := adminConn.QueryRow(ctx, "SELECT COUNT(*) FROM pg_replication_slots WHERE slot_name = ANY($1)",
				slotNames).Scan(&remainingSlotCount)
			if err != nil {
				return err
			}

			if remainingSlotCount > 0 {
				return fmt.Errorf("%d replication slots remain", remainingSlotCount)
			}

			return nil
		})
	})
}

func TestDeleteDatabaseAccess(t *testing.T) {
	postgresServer := containers.StartPostgres(t, "postgres:16")

	t.Run("lets a workspace member delete a database", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response := deleteDatabase(t, member, savedDatabase.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, tm.listDatabaseIDs(t))
	})

	t.Run("returns 403 permission_denied and keeps the database when a viewer deletes it", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response := deleteDatabase(t, viewer, savedDatabase.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
	})

	t.Run("returns 404 not_found and keeps the database when an outsider deletes another team's database",
		func(t *testing.T) {
			tm := newTeam(t)
			outsider := tm.signUpNonMember(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response := deleteDatabase(t, outsider, savedDatabase.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
			assert.Equal(t, savedDatabase.Name, tm.readDatabase(t, savedDatabase.Id).Name)
		})

	t.Run("returns 401 unauthenticated and keeps the database without a session", func(t *testing.T) {
		tm := newTeam(t)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response, err := tm.app.NewClient(t).DeleteDatabaseWithResponse(t.Context(), savedDatabase.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.Equal(t, []uuid.UUID{savedDatabase.Id}, tm.listDatabaseIDs(t))
	})
}

func deleteDatabase(t *testing.T, session apptest.Session, databaseID uuid.UUID) *apiclient.DeleteDatabaseResponse {
	t.Helper()

	response, err := session.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)

	return response
}

type logicalBackupSpec struct {
	databaseID uuid.UUID
	storageID  uuid.UUID
	status     string
}

type storedLogicalBackup struct {
	id       uuid.UUID
	fileName string
}

// insertPublishedLogicalBackup writes a backup file and its record directly, because no contract
// operation creates a backup yet. Taking the file's pending deletion away is what publishing it does.
func insertPublishedLogicalBackup(t *testing.T, tm team, spec logicalBackupSpec) storedLogicalBackup {
	t.Helper()

	storedBackup := storedLogicalBackup{id: uuid.New(), fileName: "backup-" + uuid.NewString()}

	tm.app.WriteUnpublishedFile(t, spec.storageID, storedBackup.fileName)
	tm.app.ExecuteSQL(t, "DELETE FROM storages_pending_deletions WHERE storage_id = ? AND file_name = ?",
		spec.storageID, storedBackup.fileName)
	tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{DatabaseID: spec.databaseID, StorageID: spec.storageID})
	tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, ?, ?)`, storedBackup.id, spec.databaseID, spec.storageID, spec.status, storedBackup.fileName)

	return storedBackup
}

const refusedBackupDeletionMessage = "backup deletion refused by the test"

// refuseLogicalBackupDeletion makes every deletion of a logical backup record fail, a failure the API
// cannot cause.
func refuseLogicalBackupDeletion(t *testing.T, app *apptest.App) {
	t.Helper()

	app.ExecuteSQL(t, `CREATE FUNCTION refuse_logical_backup_deletion() RETURNS trigger AS $$
		BEGIN RAISE EXCEPTION '`+refusedBackupDeletionMessage+`'; END
		$$ LANGUAGE plpgsql`)
	app.ExecuteSQL(t, `CREATE TRIGGER refuse_logical_backup_deletion BEFORE DELETE ON backups_logical
		FOR EACH ROW EXECUTE FUNCTION refuse_logical_backup_deletion()`)
}

// readReplicationSlotNames reads the slot names the module gave the database, which the API does not
// return: the WAL streamer's slot first, then the per-backup slot.
func readReplicationSlotNames(t *testing.T, tm team, databaseID uuid.UUID) []string {
	t.Helper()

	var slotRows []struct {
		ReplicationSlotName string
		BackupSlotName      string
	}
	tm.app.QuerySQL(t, &slotRows, `SELECT replication_slot_name, backup_slot_name
		FROM databases_postgresql_physical WHERE database_id = ?`, databaseID)
	require.Len(t, slotRows, 1)

	return []string{slotRows[0].ReplicationSlotName, slotRows[0].BackupSlotName}
}

// walStreamerTask stands in for the WAL supervisor's streamer: it streams from the slot and is
// registered for cancellation under the database ID, as the supervisor registers pg_receivewal.
// When cancelled, it closes its stream and reports that it stopped.
type walStreamerTask struct {
	stopped chan struct{}
}

func startWalStreamerTask(
	t *testing.T,
	server containers.Endpoint,
	databaseID uuid.UUID,
	walSlotName string,
) walStreamerTask {
	t.Helper()

	walStream := attachWalStream(t, server, walSlotName)
	streamer := walStreamerTask{stopped: make(chan struct{})}

	backups_bridge.GetCancellationRegistry().Register(databaseID, func() {
		_ = walStream.Close(t.Context())
		close(streamer.stopped)
	})
	t.Cleanup(func() { backups_bridge.GetCancellationRegistry().Unregister(databaseID) })

	return streamer
}

// attachWalStream streams from the slot over a physical replication connection, as the WAL streamer's
// pg_receivewal does, so the server reports the slot active until the stream ends.
func attachWalStream(t *testing.T, server containers.Endpoint, slotName string) *pgconn.PgConn {
	t.Helper()

	adminConn := connectAsAdmin(t, server, containers.PostgresDatabase)

	var restartLsn string
	require.NoError(t, adminConn.QueryRow(t.Context(),
		"SELECT restart_lsn::text FROM pg_replication_slots WHERE slot_name = $1", slotName).Scan(&restartLsn))

	streamConn, err := pgconn.Connect(t.Context(), fmt.Sprintf(
		"host=%s port=%d user='%s' password='%s' dbname=postgres sslmode=disable replication=true",
		server.Host, server.Port, containers.PostgresUsername, containers.PostgresPassword,
	))
	require.NoError(t, err)

	t.Cleanup(func() { _ = streamConn.Close(t.Context()) })

	frontend := streamConn.Frontend()
	frontend.Send(&pgproto3.Query{String: fmt.Sprintf("START_REPLICATION SLOT %s PHYSICAL %s", slotName, restartLsn)})
	require.NoError(t, frontend.Flush())

	for {
		message, err := frontend.Receive()
		require.NoError(t, err)

		switch typedMessage := message.(type) {
		case *pgproto3.CopyBothResponse:
			return streamConn
		case *pgproto3.ErrorResponse:
			t.Fatalf("failed to start streaming from slot %s: %s", slotName, typedMessage.Message)
		}
	}
}
