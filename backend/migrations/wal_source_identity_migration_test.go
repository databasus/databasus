package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalSourceIdentityMigration(t *testing.T) {
	t.Parallel()
	t.Run(
		"preserves registered physical databases and archives when adding the source refusal column",
		func(t *testing.T) {
			t.Parallel()
			const migrationName = "20261110170000_wal_source_identity_failure.sql"
			db := openDatabaseMigratedBefore(t, migrationName)
			workspaceID := insertWorkspace(t, db)
			databaseID := uuid.New()
			storageID := insertStorage(t, db, workspaceID)
			execute(
				t,
				db,
				"INSERT INTO databases(id,name,type,workspace_id) VALUES(?,'existing source','POSTGRES_PHYSICAL',?)",
				databaseID,
				workspaceID,
			)
			execute(
				t,
				db,
				`INSERT INTO databases_postgresql_physical(database_id,version,host,port,username,password,replication_slot_name,backup_slot_name,system_identifier,backup_type,wal_segment_size_bytes)
 VALUES(?,'17','cluster.example.com',5432,'backup','encrypted-password','existing_slot','existing_backup_slot','7706636905256173607','FULL_INCREMENTAL_WAL_STREAM',16777216)`,
				databaseID,
			)
			execute(
				t,
				db,
				"INSERT INTO backups_physical_wal_streamers(database_id,status) VALUES(?,'RUNNING')",
				databaseID,
			)
			fullID := uuid.New()
			execute(
				t,
				db,
				`INSERT INTO backups_physical(id,database_id,storage_id,timeline_id,status,kind,file_name,start_lsn,stop_lsn,completed_at)
 VALUES(?,?,?,1,'COMPLETED','FULL','original-full','0/100','0/200',NOW())`,
				fullID,
				databaseID,
				storageID,
			)
			execute(
				t,
				db,
				`INSERT INTO backups_physical_wal_segments(database_id,storage_id,timeline_id,wal_filename,start_lsn,end_lsn,file_name)
 VALUES(?,?,1,'000000010000000000000001','0/1000000','0/2000000','original-wal')`,
				databaseID,
				storageID,
			)
			var originalSettings map[string]any
			require.NoError(
				t,
				db.WithContext(t.Context()).
					Table("databases_postgresql_physical").
					Where("database_id=?", databaseID).
					Take(&originalSettings).
					Error,
			)
			applyAllMigrations(t, db, migrationName)
			var upgradedSettings map[string]any
			require.NoError(
				t,
				db.WithContext(t.Context()).
					Table("databases_postgresql_physical").
					Where("database_id=?", databaseID).
					Take(&upgradedSettings).
					Error,
			)
			require.Equal(t, originalSettings, upgradedSettings)
			var refusal *string
			require.NoError(
				t,
				db.WithContext(t.Context()).
					Raw("SELECT error_reason FROM backups_physical_wal_streamers WHERE database_id=?", databaseID).
					Scan(&refusal).
					Error,
			)
			require.Nil(t, refusal)
			require.Equal(
				t,
				[]uuid.UUID{fullID},
				listIDs(t, db, "SELECT id FROM backups_physical WHERE file_name='original-full'"),
			)
			var walFile string
			require.NoError(
				t,
				db.WithContext(t.Context()).
					Raw("SELECT file_name FROM backups_physical_wal_segments WHERE database_id=?", databaseID).
					Scan(&walFile).
					Error,
			)
			require.Equal(t, "original-wal", walFile)
		},
	)
}
