package backups_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/util/testing/containers"
)

func TestDeleteDatabaseWithPhysicalBackups(t *testing.T) {
	physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")

	t.Run(
		"removes every physical record and requests its file deletion when the database has a chain and WAL",
		func(t *testing.T) {
			tm := newTeam(t)
			physicalDatabase := tm.createPhysicalDatabase(
				t,
				physicalServer,
				apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			)
			storageID := tm.createLocalStorage(t)
			fullID := insertCompletedPhysicalFullBackup(t, tm, physicalDatabase.Id, storageID)
			fullFileName := "full-" + fullID.String()
			fileNames := []string{fullFileName, "full-manifest"}
			artifactFileNames := []string{fullFileName, "wal-first", "wal-second", "timeline-history"}
			tm.app.ExecuteSQL(
				t,
				"UPDATE backups_physical_full SET manifest_file_name = ? WHERE id = ?",
				"full-manifest",
				fullID,
			)
			var parentIncrementalID *uuid.UUID
			for range 2 {
				incrementalID := uuid.New()
				fileName := "incremental-" + incrementalID.String()
				manifestName := fileName + "-manifest"
				tm.app.ExecuteSQL(t, `INSERT INTO backups_physical_incremental
    (id, database_id, storage_id, timeline_id, status, root_full_backup_id, parent_incremental_backup_id, file_name, manifest_file_name)
    VALUES (?, ?, ?, 1, 'COMPLETED', ?, ?, ?, ?)`, incrementalID, physicalDatabase.Id, storageID, fullID, parentIncrementalID, fileName, manifestName)
				fileNames = append(fileNames, fileName, manifestName)
				artifactFileNames = append(artifactFileNames, fileName)
				parentIncrementalID = &incrementalID
			}
			for _, segment := range []struct {
				fileName string
				startLSN string
				endLSN   string
			}{
				{"wal-first", "0/1000000", "0/2000000"}, {"wal-second", "0/2000000", "0/3000000"},
			} {
				tm.app.ExecuteSQL(t, `INSERT INTO backups_physical_wal_segments
    (database_id, storage_id, timeline_id, wal_filename, file_name, start_lsn, end_lsn)
    VALUES (?, ?, 1, ?, ?, ?::pg_lsn, ?::pg_lsn)`, physicalDatabase.Id, storageID, segment.fileName, segment.fileName, segment.startLSN, segment.endLSN)
				fileNames = append(fileNames, segment.fileName)
			}
			tm.app.ExecuteSQL(t, `INSERT INTO backups_physical_wal_history_files
   (database_id, storage_id, timeline_id, file_name, history_filename)
   VALUES (?, ?, 2, 'timeline-history', '00000002.history')`, physicalDatabase.Id, storageID)
			fileNames = append(fileNames, "timeline-history")
			tm.app.ExecuteSQL(t, `INSERT INTO backups_physical_in_flight (database_id, backup_type, backup_id)
   VALUES (?, 'INCREMENTAL', ?)`, physicalDatabase.Id, uuid.New())
			tm.app.ExecuteSQL(
				t,
				`INSERT INTO backups_physical_wal_streamers (database_id, status) VALUES (?, 'RUNNING')`,
				physicalDatabase.Id,
			)
			tm.app.ExecuteSQL(t, `INSERT INTO backups_physical_wal_stream_samples
   (database_id, sampled_at, session_started_at, source_state, archived_segments, archived_wal_bytes,
   archived_stored_bytes, wal_segment_size_bytes, highest_concurrent_uploads, local_queue_bytes,
   local_queue_pause_bytes, receiver_restarts, receiver_pauses, is_receiver_running, is_forced_rotation_denied)
   VALUES (?, NOW(), NOW(), 'REACHABLE', 2, 33554432, 100, 16777216, 1, 0, 0, 0, 0, TRUE, FALSE)`, physicalDatabase.Id)

			response, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), physicalDatabase.Id)

			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
			for _, table := range []string{"backups_physical_full", "backups_physical_incremental", "backups_physical_wal_segments", "backups_physical_wal_history_files", "backups_physical_in_flight", "backups_physical_wal_streamers", "backups_physical_wal_stream_samples"} {
				assert.Zero(
					t,
					countRows(t, tm, "SELECT COUNT(*) FROM "+table+" WHERE database_id = ?", physicalDatabase.Id),
					table,
				)
			}
			for _, fileName := range fileNames {
				assert.Equal(
					t,
					1,
					countRows(
						t,
						tm,
						"SELECT COUNT(*) FROM storages_pending_deletions WHERE storage_id = ? AND file_name = ?",
						storageID,
						fileName,
					),
					fileName,
				)
			}
			for _, fileName := range artifactFileNames {
				assert.Equal(
					t,
					1,
					countRows(
						t,
						tm,
						"SELECT COUNT(*) FROM storages_pending_deletions WHERE storage_id = ? AND file_name = ?",
						storageID,
						fileName+".metadata",
					),
					fileName,
				)
			}
		},
	)
}
