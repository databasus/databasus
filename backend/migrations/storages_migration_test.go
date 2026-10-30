package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const storagesMigrationName = "20261026120000_move_storages_into_module.sql"

func TestStoragesMigration(t *testing.T) {
	t.Parallel()

	for _, storageClass := range []string{"COLD", "DEEP_ARCHIVE"} {
		t.Run("preserves the provider class when it is "+storageClass, func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, storagesMigrationName)
			workspaceID := insertWorkspace(t, db)
			storageID := insertOldStorage(t, db, "S3", &workspaceID)
			execute(t, db, `INSERT INTO s3_storages
				(storage_id, s3_bucket, s3_region, s3_access_key, s3_secret_key, s3_storage_class)
				VALUES (?, 'backups', 'us-east-1', 'enc:access', 'enc:secret', ?)`, storageID, storageClass)

			applyAllMigrations(t, db, storagesMigrationName)

			var savedClass string
			require.NoError(t, db.WithContext(t.Context()).Raw(
				"SELECT s3_storage_class FROM storages_s3 WHERE storage_id = ?", storageID,
			).Scan(&savedClass).Error)
			assert.Equal(t, storageClass, savedClass)
		})
	}

	t.Run("deletes the storages that cannot work and the provider rows of another type", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, storagesMigrationName)
		workspaceID := insertWorkspace(t, db)
		databaseID := insertDatabase(t, db, &workspaceID)

		retypedStorageID := insertOldStorage(t, db, "LOCAL", &workspaceID)
		execute(t, db, "INSERT INTO local_storages (storage_id) VALUES (?)", retypedStorageID)
		execute(t, db, `INSERT INTO s3_storages (storage_id, s3_bucket, s3_region, s3_access_key, s3_secret_key)
			VALUES (?, 'left-behind', 'us-east-1', 'enc:access', 'enc:secret')`, retypedStorageID)

		unknownTypeStorageID := insertOldStorage(t, db, "DROPBOX", &workspaceID)
		unusedStorageWithoutRowID := insertOldStorage(t, db, "FTP", &workspaceID)
		usedStorageWithoutRowID := insertOldStorage(t, db, "SFTP", &workspaceID)
		insertOldBackupConfig(t, db, databaseID, usedStorageWithoutRowID)

		applyAllMigrations(t, db, storagesMigrationName)

		assert.ElementsMatch(t,
			[]uuid.UUID{retypedStorageID, usedStorageWithoutRowID},
			listIDs(t, db, "SELECT id FROM storages"))
		assert.ElementsMatch(t, []uuid.UUID{retypedStorageID}, listIDs(t, db, "SELECT storage_id FROM storages_local"))
		assert.Empty(t, listIDs(t, db, "SELECT storage_id FROM storages_s3"))
		assert.NotContains(t, listIDs(t, db, "SELECT id FROM storages"), unknownTypeStorageID)
		assert.NotContains(t, listIDs(t, db, "SELECT id FROM storages"), unusedStorageWithoutRowID)
		assert.ElementsMatch(t, []uuid.UUID{usedStorageWithoutRowID},
			listIDs(t, db, "SELECT storage_id FROM backups_configs"))
	})

	t.Run("moves a storage without a workspace into its database's workspace and an unused one into the oldest",
		func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, storagesMigrationName)
			oldestWorkspaceID := insertWorkspace(t, db)
			execute(t, db, "UPDATE workspaces SET created_at = '2020-01-01' WHERE id = ?", oldestWorkspaceID)
			databaseWorkspaceID := insertWorkspace(t, db)
			databaseID := insertDatabase(t, db, &databaseWorkspaceID)

			usedStorageID := insertOldStorage(t, db, "LOCAL", nil)
			execute(t, db, "INSERT INTO local_storages (storage_id) VALUES (?)", usedStorageID)
			insertOldBackupConfig(t, db, databaseID, usedStorageID)

			unusedStorageID := insertOldStorage(t, db, "LOCAL", nil)
			execute(t, db, "INSERT INTO local_storages (storage_id) VALUES (?)", unusedStorageID)

			applyAllMigrations(t, db, storagesMigrationName)

			assert.Equal(t, databaseWorkspaceID, getStorageWorkspaceID(t, db, usedStorageID))
			assert.Equal(t, oldestWorkspaceID, getStorageWorkspaceID(t, db, unusedStorageID))
		})

	t.Run("refuses rows that break the new constraints after the upgrade", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, storagesMigrationName)
		workspaceID := insertWorkspace(t, db)

		applyAllMigrations(t, db, storagesMigrationName)

		localStorageID := uuid.New()
		execute(t, db, "INSERT INTO storages (id, type, name, workspace_id) VALUES (?, 'LOCAL', 'local', ?)",
			localStorageID, workspaceID)

		refusedStatements := []struct {
			name       string
			statement  string
			values     []any
			constraint string
		}{
			{
				name:       "storage without a workspace",
				statement:  "INSERT INTO storages (id, type, name) VALUES (?, 'LOCAL', 'orphan')",
				values:     []any{uuid.New()},
				constraint: "workspace_id",
			},
			{
				name: "provider row of another type than its storage",
				statement: `INSERT INTO storages_s3 (storage_id, s3_bucket, s3_region, s3_access_key, s3_secret_key)
					VALUES (?, 'bucket', 'us-east-1', 'enc:access', 'enc:secret')`,
				values:     []any{localStorageID},
				constraint: "fk_storages_s3_storage",
			},
		}

		for _, refusedStatement := range refusedStatements {
			statementResult := db.WithContext(t.Context()).Exec(refusedStatement.statement, refusedStatement.values...)
			require.Error(t, statementResult.Error, refusedStatement.name)
			assert.Contains(t, statementResult.Error.Error(), refusedStatement.constraint, refusedStatement.name)
		}

		var lastSaveErrorColumns []string
		require.NoError(t, db.WithContext(t.Context()).Raw(
			`SELECT column_name FROM information_schema.columns
			WHERE table_name = 'storages' AND column_name = 'last_save_error'`,
		).Scan(&lastSaveErrorColumns).Error)
		assert.Empty(t, lastSaveErrorColumns)
	})

	t.Run("keeps every encrypted secret byte for byte", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, storagesMigrationName)
		workspaceID := insertWorkspace(t, db)
		secretColumns := insertOldStorageOfEveryTypeWithSecrets(t, db, workspaceID)

		applyAllMigrations(t, db, storagesMigrationName)

		for _, secretColumn := range secretColumns {
			var storedValues []string
			require.NoError(t, db.WithContext(t.Context()).Raw(
				"SELECT "+secretColumn.column+" FROM "+secretColumn.table+" WHERE storage_id = ?",
				secretColumn.storageID,
			).Scan(&storedValues).Error)
			assert.Equal(
				t,
				[]string{secretColumn.value},
				storedValues,
				"%s.%s",
				secretColumn.table,
				secretColumn.column,
			)
		}
	})
}

type storedStorageSecret struct {
	storageID uuid.UUID
	table     string
	column    string
	value     string
}

func insertOldStorageOfEveryTypeWithSecrets(
	t *testing.T,
	db *gorm.DB,
	workspaceID uuid.UUID,
) []storedStorageSecret {
	t.Helper()

	s3StorageID := insertOldStorage(t, db, "S3", &workspaceID)
	execute(t, db, `INSERT INTO s3_storages (storage_id, s3_bucket, s3_region, s3_access_key, s3_secret_key)
		VALUES (?, 'backups', 'us-east-1', 'enc:s3-access', 'enc:s3-secret')`, s3StorageID)

	googleDriveStorageID := insertOldStorage(t, db, "GOOGLE_DRIVE", &workspaceID)
	execute(t, db, `INSERT INTO google_drive_storages (storage_id, client_id, client_secret, token_json)
		VALUES (?, 'client', 'enc:drive-secret', 'enc:drive-token')`, googleDriveStorageID)

	nasStorageID := insertOldStorage(t, db, "NAS", &workspaceID)
	execute(t, db, `INSERT INTO nas_storages (storage_id, host, port, share, username, password)
		VALUES (?, 'nas.example.com', 445, 'backups', 'ops', 'enc:nas-password')`, nasStorageID)

	azureBlobStorageID := insertOldStorage(t, db, "AZURE_BLOB", &workspaceID)
	execute(t, db, `INSERT INTO azure_blob_storages
		(storage_id, auth_method, connection_string, account_name, account_key, container_name)
		VALUES (?, 'ACCOUNT_KEY', 'enc:azure-connection', 'account', 'enc:azure-key', 'backups')`,
		azureBlobStorageID)

	ftpStorageID := insertOldStorage(t, db, "FTP", &workspaceID)
	execute(t, db, `INSERT INTO ftp_storages (storage_id, host, port, username, password)
		VALUES (?, 'ftp.example.com', 21, 'ops', 'enc:ftp-password')`, ftpStorageID)

	sftpStorageID := insertOldStorage(t, db, "SFTP", &workspaceID)
	execute(t, db, `INSERT INTO sftp_storages (storage_id, host, port, username, password, private_key)
		VALUES (?, 'sftp.example.com', 22, 'ops', 'enc:sftp-password', 'enc:sftp-key')`, sftpStorageID)

	rcloneStorageID := insertOldStorage(t, db, "RCLONE", &workspaceID)
	execute(t, db, "INSERT INTO rclone_storages (storage_id, config_content) VALUES (?, 'enc:rclone-config')",
		rcloneStorageID)

	return []storedStorageSecret{
		{storageID: s3StorageID, table: "storages_s3", column: "s3_access_key", value: "enc:s3-access"},
		{storageID: s3StorageID, table: "storages_s3", column: "s3_secret_key", value: "enc:s3-secret"},
		{
			storageID: googleDriveStorageID, table: "storages_google_drive", column: "client_secret",
			value: "enc:drive-secret",
		},
		{
			storageID: googleDriveStorageID, table: "storages_google_drive", column: "token_json",
			value: "enc:drive-token",
		},
		{storageID: nasStorageID, table: "storages_nas", column: "password", value: "enc:nas-password"},
		{
			storageID: azureBlobStorageID, table: "storages_azure_blob", column: "connection_string",
			value: "enc:azure-connection",
		},
		{storageID: azureBlobStorageID, table: "storages_azure_blob", column: "account_key", value: "enc:azure-key"},
		{storageID: ftpStorageID, table: "storages_ftp", column: "password", value: "enc:ftp-password"},
		{storageID: sftpStorageID, table: "storages_sftp", column: "password", value: "enc:sftp-password"},
		{storageID: sftpStorageID, table: "storages_sftp", column: "private_key", value: "enc:sftp-key"},
		{storageID: rcloneStorageID, table: "storages_rclone", column: "config_content", value: "enc:rclone-config"},
	}
}

func insertOldStorage(t *testing.T, db *gorm.DB, storageType string, workspaceID *uuid.UUID) uuid.UUID {
	t.Helper()

	storageID := uuid.New()
	execute(t, db, "INSERT INTO storages (id, type, name, workspace_id) VALUES (?, ?, ?, ?)",
		storageID, storageType, "storage "+storageID.String(), workspaceID)

	return storageID
}

func insertOldBackupConfig(t *testing.T, db *gorm.DB, databaseID, storageID uuid.UUID) {
	t.Helper()

	execute(t, db, `INSERT INTO logical_backup_configs
		(database_id, storage_id, send_notifications_on, interval_type) VALUES (?, ?, '', 'DAILY')`,
		databaseID, storageID)
}

func getStorageWorkspaceID(t *testing.T, db *gorm.DB, storageID uuid.UUID) uuid.UUID {
	t.Helper()

	var workspaceIDs []uuid.UUID
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT workspace_id FROM storages WHERE id = ?", storageID,
	).Scan(&workspaceIDs).Error)
	require.Len(t, workspaceIDs, 1)

	return workspaceIDs[0]
}
