package migrations_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/migrations"
)

const databasesMigrationName = "20261027090000_move_databases_into_module.sql"

func TestDatabasesMigration(t *testing.T) {
	t.Parallel()

	t.Run("deletes the provider rows that are never read and the databases that cannot work", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := insertStorage(t, db, workspaceID)

		logicalDatabaseID := insertDatabase(t, db, &workspaceID)
		execute(t, db, `INSERT INTO mongodb_databases (database_id, version, host, username, password, database)
			VALUES (?, '7', 'mongo.example.com', 'backup', 'enc:stray', 'shop')`, logicalDatabaseID)
		execute(t, db, `INSERT INTO postgresql_logical_databases (database_id, version, host, port, username, password)
			VALUES (NULL, '16', 'orphan.example.com', 5432, 'backup', 'enc:orphan')`)
		execute(t, db, `INSERT INTO postgresql_logical_databases (database_id, version, host, port, username, password)
			VALUES (?, '16', 'orphan.example.com', 5432, 'backup', 'enc:orphan')`, uuid.New())

		mysqlDatabaseID := insertOldDatabase(t, db, "MYSQL", &workspaceID)
		keptMysqlRowID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		insertOldMysqlRow(t, db, mysqlDatabaseID, keptMysqlRowID, "kept.example.com")
		insertOldMysqlRow(
			t,
			db,
			mysqlDatabaseID,
			uuid.MustParse("ffffffff-0000-0000-0000-000000000001"),
			"dropped.example.com",
		)

		unusedDatabaseWithoutRowID := insertOldDatabase(t, db, "MARIADB", &workspaceID)
		usedDatabaseWithoutRowID := insertOldDatabase(t, db, "MARIADB", &workspaceID)
		insertOldBackupConfig(t, db, usedDatabaseWithoutRowID, storageID)
		unknownTypeDatabaseID := insertOldDatabase(t, db, "ORACLE", &workspaceID)

		applyAllMigrations(t, db, databasesMigrationName)

		assert.ElementsMatch(t,
			[]uuid.UUID{logicalDatabaseID, mysqlDatabaseID, usedDatabaseWithoutRowID},
			listIDs(t, db, "SELECT id FROM databases"))
		assert.ElementsMatch(t, []uuid.UUID{logicalDatabaseID},
			listIDs(t, db, "SELECT database_id FROM databases_postgresql_logical"))
		assert.Empty(t, listIDs(t, db, "SELECT database_id FROM databases_mongodb"))
		assert.Equal(t, []string{"kept.example.com"},
			listStrings(t, db, "SELECT host FROM databases_mysql WHERE database_id = ?", mysqlDatabaseID))
		assert.NotContains(t, listIDs(t, db, "SELECT id FROM databases"), unusedDatabaseWithoutRowID)
		assert.NotContains(t, listIDs(t, db, "SELECT id FROM databases"), unknownTypeDatabaseID)
	})

	t.Run("moves a database without a workspace into the oldest one when backups reference it", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		oldestWorkspaceID := insertWorkspace(t, db)
		execute(t, db, "UPDATE workspaces SET created_at = '2020-01-01' WHERE id = ?", oldestWorkspaceID)
		newerWorkspaceID := insertWorkspace(t, db)
		storageID := insertStorage(t, db, newerWorkspaceID)

		databaseWithBackupsID := insertDatabase(t, db, nil)
		execute(t, db, `INSERT INTO logical_backups (id, database_id, storage_id, status, file_name)
			VALUES (?, ?, ?, 'COMPLETED', 'backup.dump')`, uuid.New(), databaseWithBackupsID, storageID)
		databaseWithNothingID := insertDatabase(t, db, nil)

		applyAllMigrations(t, db, databasesMigrationName)

		assert.Equal(t, []uuid.UUID{oldestWorkspaceID},
			listIDs(t, db, "SELECT workspace_id FROM databases WHERE id = '"+databaseWithBackupsID.String()+"'"))
		assert.NotContains(t, listIDs(t, db, "SELECT id FROM databases"), databaseWithNothingID)
	})

	t.Run("converts every list column the way the old code read it", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)
		rawList := "  first ,second\n\tthird\r\n,, first,\t \nsecond, fourth "
		convertedList := []string{"first", "second", "third", "fourth"}

		logicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_LOGICAL", &workspaceID)
		execute(t, db, `INSERT INTO postgresql_logical_databases
			(database_id, version, host, port, username, password, include_schemas, exclude_tables, exclude_table_data)
			VALUES (?, '16', 'db.example.com', 5432, 'backup', 'enc:p', ?, ?, ?)`,
			logicalDatabaseID, rawList, rawList, rawList)
		mysqlDatabaseID := insertOldDatabase(t, db, "MYSQL", &workspaceID)
		execute(t, db, `INSERT INTO mysql_databases (database_id, version, host, port, username, password,
			exclude_tables, privileges) VALUES (?, '8.0', 'db.example.com', 3306, 'backup', 'enc:p', ?, ?)`,
			mysqlDatabaseID, rawList, "SELECT,SHOW VIEW")
		mariadbDatabaseID := insertOldDatabase(t, db, "MARIADB", &workspaceID)
		execute(t, db, `INSERT INTO mariadb_databases (database_id, version, host, port, username, password,
			exclude_tables, privileges) VALUES (?, '10.11', 'db.example.com', 3306, 'backup', 'enc:p', ?, ?)`,
			mariadbDatabaseID, rawList, rawList)
		mongodbDatabaseID := insertOldDatabase(t, db, "MONGODB", &workspaceID)
		execute(t, db, `INSERT INTO mongodb_databases (database_id, version, host, port, username, password, database,
			exclude_collections) VALUES (?, '7', 'mongo.example.com', 27017, 'backup', 'enc:p', 'shop', ?)`,
			mongodbDatabaseID, rawList)
		emptyListDatabaseID := insertDatabase(t, db, &workspaceID)

		applyAllMigrations(t, db, databasesMigrationName)

		for _, listColumn := range []struct {
			table      string
			column     string
			databaseID uuid.UUID
			expected   []string
		}{
			{"databases_postgresql_logical", "include_schemas", logicalDatabaseID, convertedList},
			{"databases_postgresql_logical", "exclude_tables", logicalDatabaseID, convertedList},
			{"databases_postgresql_logical", "exclude_table_data", logicalDatabaseID, convertedList},
			{"databases_postgresql_logical", "exclude_tables", emptyListDatabaseID, []string{}},
			{"databases_mysql", "exclude_tables", mysqlDatabaseID, convertedList},
			{"databases_mysql", "privileges", mysqlDatabaseID, []string{"SELECT", "SHOW VIEW"}},
			{"databases_mariadb", "exclude_tables", mariadbDatabaseID, convertedList},
			{"databases_mariadb", "privileges", mariadbDatabaseID, convertedList},
			{"databases_mongodb", "exclude_collections", mongodbDatabaseID, convertedList},
		} {
			assert.Equal(t, listColumn.expected,
				getList(t, db, listColumn.table, listColumn.column, listColumn.databaseID),
				listColumn.table+"."+listColumn.column)
		}
	})

	t.Run("sets the last backup time of a physical database to its newest backup", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := insertStorage(t, db, workspaceID)
		fullBackupTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		incrementalBackupTime := fullBackupTime.Add(time.Hour)
		walSegmentTime := fullBackupTime.Add(2 * time.Hour)

		physicalDatabaseID := insertOldPhysicalDatabase(t, db, workspaceID)
		fullBackupID := uuid.New()
		insertPhysicalBackup(t, db, physicalBackup{
			table: "physical_full_backups", id: fullBackupID, databaseID: physicalDatabaseID, storageID: storageID,
			status: "COMPLETED", createdAt: fullBackupTime,
		})
		insertPhysicalBackup(t, db, physicalBackup{
			table: "physical_full_backups", id: uuid.New(), databaseID: physicalDatabaseID, storageID: storageID,
			status: "FAILED", createdAt: walSegmentTime.Add(time.Hour),
		})
		execute(t, db, `INSERT INTO physical_incremental_backups
			(id, database_id, storage_id, timeline_id, status, created_at, root_full_backup_id)
			VALUES (?, ?, ?, 1, 'COMPLETED', ?, ?)`,
			uuid.New(), physicalDatabaseID, storageID, incrementalBackupTime, fullBackupID)
		insertWalSegment(t, db, physicalDatabaseID, storageID, walSegmentTime, "segment-1")
		insertWalSegment(t, db, physicalDatabaseID, storageID, walSegmentTime.Add(time.Hour), "")

		databaseWithFullBackupOnlyID := insertOldPhysicalDatabase(t, db, workspaceID)
		insertPhysicalBackup(t, db, physicalBackup{
			table: "physical_full_backups", id: uuid.New(), databaseID: databaseWithFullBackupOnlyID,
			storageID: storageID, status: "COMPLETED", createdAt: fullBackupTime,
		})

		applyAllMigrations(t, db, databasesMigrationName)

		assert.Equal(t, walSegmentTime, getLastBackupTime(t, db, physicalDatabaseID))
		assert.Equal(t, fullBackupTime, getLastBackupTime(t, db, databaseWithFullBackupOnlyID))
	})

	t.Run("keeps the name of the per-backup slot a physical database already uses", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)
		physicalDatabaseID := insertOldPhysicalDatabase(t, db, workspaceID)

		var oldRowID string
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT id::text FROM postgresql_physical_databases WHERE database_id = ?", physicalDatabaseID,
		).Row().Scan(&oldRowID))

		applyAllMigrations(t, db, databasesMigrationName)

		var backupSlotName string
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT backup_slot_name FROM databases_postgresql_physical WHERE database_id = ?", physicalDatabaseID,
		).Scan(&backupSlotName).Error)
		assert.Equal(t, "databasus_basebackup_"+strings.ReplaceAll(oldRowID, "-", ""), backupSlotName)
	})

	t.Run("keeps every encrypted secret byte for byte", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)
		secretColumns := insertOldDatabaseOfEveryTypeWithSecrets(t, db, workspaceID)

		applyAllMigrations(t, db, databasesMigrationName)

		for _, secretColumn := range secretColumns {
			assert.Equal(t, []string{secretColumn.value},
				listStrings(t, db,
					"SELECT "+secretColumn.column+" FROM "+secretColumn.table+" WHERE database_id = ?",
					secretColumn.databaseID),
				secretColumn.table+"."+secretColumn.column)
		}
	})

	t.Run("refuses rows that break the new constraints after the upgrade", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)

		applyAllMigrations(t, db, databasesMigrationName)

		logicalDatabaseID := insertMigratedDatabase(t, db, &workspaceID)

		for _, refusedStatement := range []struct {
			name       string
			statement  string
			values     []any
			constraint string
		}{
			{
				name:       "database without a workspace",
				statement:  "INSERT INTO databases (id, name, type) VALUES (?, 'x', 'POSTGRES_LOGICAL')",
				values:     []any{uuid.New()},
				constraint: "workspace_id",
			},
			{
				name: "provider row of another type than its database",
				statement: `INSERT INTO databases_mysql (database_id, version, host, port, username, password)
					VALUES (?, '8.0', 'db.example.com', 3306, 'backup', 'enc:p')`,
				values:     []any{logicalDatabaseID},
				constraint: "fk_databases_mysql_database",
			},
			{
				name: "second settings row of one database",
				statement: `INSERT INTO databases_postgresql_logical (database_id, version, host, port, username, password)
					VALUES (?, '16', 'db.example.com', 5432, 'backup', 'enc:p'),
					(?, '16', 'db.example.com', 5432, 'backup', 'enc:p')`,
				values:     []any{logicalDatabaseID, logicalDatabaseID},
				constraint: "databases_postgresql_logical_pkey",
			},
		} {
			statementResult := db.WithContext(t.Context()).Exec(refusedStatement.statement, refusedStatement.values...)
			require.Error(t, statementResult.Error, refusedStatement.name)
			assert.Contains(t, statementResult.Error.Error(), refusedStatement.constraint, refusedStatement.name)
		}
	})

	t.Run("keys every settings table by the database it belongs to", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)
		secretColumns := insertOldDatabaseOfEveryTypeWithSecrets(t, db, workspaceID)

		applyAllMigrations(t, db, databasesMigrationName)

		for _, settingsTable := range []string{
			"databases_postgresql_logical",
			"databases_postgresql_physical",
			"databases_mysql",
			"databases_mariadb",
			"databases_mongodb",
		} {
			assert.Equal(t, []string{"database_id"}, listStrings(t, db, `SELECT a.attname
				FROM pg_index i
				JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
				WHERE i.indrelid = ?::regclass AND i.indisprimary`, settingsTable), settingsTable)
			assert.Empty(t, listStrings(t, db, `SELECT column_name FROM information_schema.columns
				WHERE table_name = ? AND column_name = 'id'`, settingsTable), settingsTable)
		}

		for _, secretColumn := range secretColumns {
			assert.Equal(t, []string{secretColumn.databaseID.String()},
				listStrings(t, db, "SELECT database_id::text FROM "+secretColumn.table+" WHERE database_id = ?",
					secretColumn.databaseID),
				secretColumn.table)
		}
	})

	t.Run("stops with the database's ID when a database of an unknown type has backups", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, databasesMigrationName)
		workspaceID := insertWorkspace(t, db)
		storageID := insertStorage(t, db, workspaceID)
		unknownTypeDatabaseID := insertOldDatabase(t, db, "ORACLE", &workspaceID)
		insertOldBackupConfig(t, db, unknownTypeDatabaseID, storageID)

		_, err := database.Migrate(t.Context(), db, migrations.Files)

		require.Error(t, err)
		assert.Contains(t, err.Error(), unknownTypeDatabaseID.String())
		assert.Contains(t, listIDs(t, db, "SELECT id FROM databases"), unknownTypeDatabaseID)
	})
}

type secretColumn struct {
	databaseID uuid.UUID
	table      string
	column     string
	value      string
}

func insertOldDatabaseOfEveryTypeWithSecrets(t *testing.T, db *gorm.DB, workspaceID uuid.UUID) []secretColumn {
	t.Helper()

	logicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_LOGICAL", &workspaceID)
	execute(t, db, `INSERT INTO postgresql_logical_databases (database_id, version, host, port, username, password,
		ssl_mode, ssl_client_cert, ssl_client_key, ssl_root_cert,
		ssh_is_enabled, ssh_host, ssh_username, ssh_password, ssh_private_key, ssh_private_key_passphrase)
		VALUES (?, '16', 'db.example.com', 5432, 'backup', 'enc:logical-password',
		'verify-full', 'enc:logical-cert', 'enc:logical-key', 'enc:logical-root',
		TRUE, 'bastion', 'tunnel', 'enc:logical-ssh-password', 'enc:logical-ssh-key', 'enc:logical-ssh-passphrase')`,
		logicalDatabaseID)

	physicalDatabaseID := insertOldDatabase(t, db, "POSTGRES_PHYSICAL", &workspaceID)
	execute(t, db, `INSERT INTO postgresql_physical_databases (database_id, version, host, port, username, password,
		replication_slot_name, ssl_mode, ssl_client_cert, ssl_client_key, ssl_root_cert,
		ssh_password, ssh_private_key, ssh_private_key_passphrase)
		VALUES (?, '17', 'db.example.com', 5432, 'backup', 'enc:physical-password', 'slot',
		'verify-full', 'enc:physical-cert', 'enc:physical-key', 'enc:physical-root',
		'enc:physical-ssh-password', 'enc:physical-ssh-key', 'enc:physical-ssh-passphrase')`, physicalDatabaseID)

	mysqlDatabaseID := insertOldDatabase(t, db, "MYSQL", &workspaceID)
	execute(t, db, `INSERT INTO mysql_databases (database_id, version, host, port, username, password,
		ssh_password, ssh_private_key, ssh_private_key_passphrase)
		VALUES (?, '8.0', 'db.example.com', 3306, 'backup', 'enc:mysql-password',
		'enc:mysql-ssh-password', 'enc:mysql-ssh-key', 'enc:mysql-ssh-passphrase')`, mysqlDatabaseID)

	mariadbDatabaseID := insertOldDatabase(t, db, "MARIADB", &workspaceID)
	execute(t, db, `INSERT INTO mariadb_databases (database_id, version, host, port, username, password,
		ssh_password, ssh_private_key, ssh_private_key_passphrase)
		VALUES (?, '10.11', 'db.example.com', 3306, 'backup', 'enc:mariadb-password',
		'enc:mariadb-ssh-password', 'enc:mariadb-ssh-key', 'enc:mariadb-ssh-passphrase')`, mariadbDatabaseID)

	mongodbDatabaseID := insertOldDatabase(t, db, "MONGODB", &workspaceID)
	execute(t, db, `INSERT INTO mongodb_databases (database_id, version, host, port, username, password, database,
		ssh_password, ssh_private_key, ssh_private_key_passphrase)
		VALUES (?, '7', 'mongo.example.com', 27017, 'backup', 'enc:mongodb-password', 'shop',
		'enc:mongodb-ssh-password', 'enc:mongodb-ssh-key', 'enc:mongodb-ssh-passphrase')`, mongodbDatabaseID)

	secretColumns := []secretColumn{
		{logicalDatabaseID, "databases_postgresql_logical", "ssl_client_cert", "enc:logical-cert"},
		{logicalDatabaseID, "databases_postgresql_logical", "ssl_client_key", "enc:logical-key"},
		{logicalDatabaseID, "databases_postgresql_logical", "ssl_root_cert", "enc:logical-root"},
		{physicalDatabaseID, "databases_postgresql_physical", "ssl_client_cert", "enc:physical-cert"},
		{physicalDatabaseID, "databases_postgresql_physical", "ssl_client_key", "enc:physical-key"},
		{physicalDatabaseID, "databases_postgresql_physical", "ssl_root_cert", "enc:physical-root"},
	}

	for _, typedDatabase := range []struct {
		databaseID uuid.UUID
		table      string
		prefix     string
	}{
		{logicalDatabaseID, "databases_postgresql_logical", "logical"},
		{physicalDatabaseID, "databases_postgresql_physical", "physical"},
		{mysqlDatabaseID, "databases_mysql", "mysql"},
		{mariadbDatabaseID, "databases_mariadb", "mariadb"},
		{mongodbDatabaseID, "databases_mongodb", "mongodb"},
	} {
		secretColumns = append(
			secretColumns,
			secretColumn{
				typedDatabase.databaseID,
				typedDatabase.table,
				"password",
				"enc:" + typedDatabase.prefix + "-password",
			},
			secretColumn{
				typedDatabase.databaseID, typedDatabase.table, "ssh_password",
				"enc:" + typedDatabase.prefix + "-ssh-password",
			},
			secretColumn{
				typedDatabase.databaseID, typedDatabase.table, "ssh_private_key",
				"enc:" + typedDatabase.prefix + "-ssh-key",
			},
			secretColumn{
				typedDatabase.databaseID, typedDatabase.table, "ssh_private_key_passphrase",
				"enc:" + typedDatabase.prefix + "-ssh-passphrase",
			},
		)
	}

	return secretColumns
}

func insertOldDatabase(t *testing.T, db *gorm.DB, databaseType string, workspaceID *uuid.UUID) uuid.UUID {
	t.Helper()

	databaseID := uuid.New()
	execute(t, db, "INSERT INTO databases (id, name, type, workspace_id) VALUES (?, ?, ?, ?)",
		databaseID, "database "+databaseID.String(), databaseType, workspaceID)

	return databaseID
}

func insertOldMysqlRow(t *testing.T, db *gorm.DB, databaseID, rowID uuid.UUID, host string) {
	t.Helper()

	execute(t, db, `INSERT INTO mysql_databases (id, database_id, version, host, port, username, password)
		VALUES (?, ?, '8.0', ?, 3306, 'backup', 'enc:p')`, rowID, databaseID, host)
}

func insertOldPhysicalDatabase(t *testing.T, db *gorm.DB, workspaceID uuid.UUID) uuid.UUID {
	t.Helper()

	databaseID := insertOldDatabase(t, db, "POSTGRES_PHYSICAL", &workspaceID)
	execute(t, db, `INSERT INTO postgresql_physical_databases
		(database_id, version, host, port, username, password, replication_slot_name)
		VALUES (?, '17', 'db.example.com', 5432, 'replicator', 'enc:p', ?)`, databaseID, "slot_"+databaseID.String())

	return databaseID
}

type physicalBackup struct {
	table      string
	id         uuid.UUID
	databaseID uuid.UUID
	storageID  uuid.UUID
	status     string
	createdAt  time.Time
}

func insertPhysicalBackup(t *testing.T, db *gorm.DB, backup physicalBackup) {
	t.Helper()

	execute(t, db, "INSERT INTO "+backup.table+` (id, database_id, storage_id, timeline_id, status, created_at)
		VALUES (?, ?, ?, 1, ?, ?)`, backup.id, backup.databaseID, backup.storageID, backup.status, backup.createdAt)
}

// insertWalSegment leaves the segment uncommitted when fileName is empty.
func insertWalSegment(
	t *testing.T,
	db *gorm.DB,
	databaseID, storageID uuid.UUID,
	receivedAt time.Time,
	fileName string,
) {
	t.Helper()

	var committedFileName *string
	if fileName != "" {
		committedFileName = &fileName
	}

	execute(t, db, `INSERT INTO physical_wal_segments
		(database_id, storage_id, timeline_id, file_name, wal_filename, start_lsn, end_lsn, received_at)
		VALUES (?, ?, 1, ?, ?, ?::pg_lsn, ?::pg_lsn, ?)`,
		databaseID, storageID, committedFileName, "wal-"+uuid.NewString(),
		"0/"+receivedAt.Format("150405"), "0/FFFFFFFF", receivedAt)
}

func getLastBackupTime(t *testing.T, db *gorm.DB, databaseID uuid.UUID) time.Time {
	t.Helper()

	var lastBackupTimes []time.Time
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT last_backup_time FROM databases WHERE id = ?", databaseID,
	).Scan(&lastBackupTimes).Error)
	require.Len(t, lastBackupTimes, 1)

	return lastBackupTimes[0].UTC()
}

func getList(t *testing.T, db *gorm.DB, table, column string, databaseID uuid.UUID) []string {
	t.Helper()

	var lists []pq.StringArray
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT "+column+" FROM "+table+" WHERE database_id = ?", databaseID,
	).Scan(&lists).Error)
	require.Len(t, lists, 1)

	return []string(lists[0])
}

func listStrings(t *testing.T, db *gorm.DB, query string, values ...any) []string {
	t.Helper()

	var listedValues []string
	require.NoError(t, db.WithContext(t.Context()).Raw(query, values...).Scan(&listedValues).Error)

	return listedValues
}
