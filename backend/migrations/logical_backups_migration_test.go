package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const logicalBackupsMigrationName = "20261027140000_move_logical_backups_into_module.sql"

func TestLogicalBackupsMigration(t *testing.T) {
	t.Parallel()

	t.Run("fails unknown statuses and keeps the stored value in the fail message", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, logicalBackupsMigrationName)
		fixture := insertLogicalBackupsFixture(t, db)
		backupID := insertOldLogicalBackup(t, db, fixture, oldLogicalBackup{status: "PAUSED", encryption: "NONE"})
		restoreID := insertOldRestore(t, db, backupID, "WAITING")

		applyAllMigrations(t, db, logicalBackupsMigrationName)

		assert.Equal(t,
			statusRow{Status: "FAILED", FailMessage: new("The upgrade replaced the unknown status PAUSED")},
			readStatusRow(t, db, "backups_logical", backupID))
		assert.Equal(t,
			statusRow{Status: "FAILED", FailMessage: new("The upgrade replaced the unknown status WAITING")},
			readStatusRow(t, db, "backups_restores", restoreID))
	})

	t.Run("repairs an unknown encryption by whether the backup has a salt", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, logicalBackupsMigrationName)
		fixture := insertLogicalBackupsFixture(t, db)
		plainBackupID := insertOldLogicalBackup(
			t,
			db,
			fixture,
			oldLogicalBackup{status: "COMPLETED", encryption: "AES"},
		)
		saltedBackupID := insertOldLogicalBackup(t, db, fixture, oldLogicalBackup{
			status:         "COMPLETED",
			encryption:     "AES",
			encryptionSalt: new("c2FsdA=="),
		})
		unknownVerificationBackupID := insertOldLogicalBackup(t, db, fixture, oldLogicalBackup{
			status:             "COMPLETED",
			encryption:         "NONE",
			verificationStatus: "MAYBE",
		})

		applyAllMigrations(t, db, logicalBackupsMigrationName)

		assert.Equal(t, []string{"NONE"},
			listStrings(t, db, "SELECT encryption FROM backups_logical WHERE id = ?", plainBackupID))
		assert.Equal(t, []string{"ENCRYPTED"},
			listStrings(t, db, "SELECT encryption FROM backups_logical WHERE id = ?", saltedBackupID))
		assert.Equal(t, []string{"NOT_VERIFIED"}, listStrings(t, db,
			"SELECT restore_verification_status FROM backups_logical WHERE id = ?", unknownVerificationBackupID))
	})

	t.Run("fails all but the newest in-progress backup and restore of a database", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, logicalBackupsMigrationName)
		fixture := insertLogicalBackupsFixture(t, db)
		olderBackupID := insertOldLogicalBackup(t, db, fixture, oldLogicalBackup{
			status:     "IN_PROGRESS",
			encryption: "NONE",
			createdAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})
		newerBackupID := insertOldLogicalBackup(t, db, fixture, oldLogicalBackup{
			status:     "IN_PROGRESS",
			encryption: "NONE",
			createdAt:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		})
		olderRestoreID := insertOldRestore(t, db, olderBackupID, "IN_PROGRESS")
		newerRestoreID := insertOldRestore(t, db, newerBackupID, "IN_PROGRESS")
		execute(t, db, "UPDATE restores SET created_at = '2026-01-02' WHERE id = ?", olderRestoreID)
		execute(t, db, "UPDATE restores SET created_at = '2026-01-03' WHERE id = ?", newerRestoreID)

		applyAllMigrations(t, db, logicalBackupsMigrationName)

		assert.Equal(t,
			statusRow{Status: "FAILED", FailMessage: new("Backup failed due to application restart")},
			readStatusRow(t, db, "backups_logical", olderBackupID))
		assert.Equal(t, statusRow{Status: "IN_PROGRESS", FailMessage: nil},
			readStatusRow(t, db, "backups_logical", newerBackupID))
		assert.Equal(t,
			statusRow{Status: "FAILED", FailMessage: new("Restore failed due to application restart")},
			readStatusRow(t, db, "backups_restores", olderRestoreID))
		assert.Equal(t, statusRow{Status: "IN_PROGRESS", FailMessage: nil},
			readStatusRow(t, db, "backups_restores", newerRestoreID))
	})

	t.Run("copies the backup's database into the restore and renames the token's used", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, logicalBackupsMigrationName)
		fixture := insertLogicalBackupsFixture(t, db)
		backupID := insertOldLogicalBackup(t, db, fixture, oldLogicalBackup{status: "COMPLETED", encryption: "NONE"})
		restoreID := insertOldRestore(t, db, backupID, "COMPLETED")
		usedTokenID := insertOldDownloadToken(t, db, backupID, fixture.userID, true)
		unusedTokenID := insertOldDownloadToken(t, db, backupID, fixture.userID, false)

		applyAllMigrations(t, db, logicalBackupsMigrationName)

		assert.Equal(t, []uuid.UUID{fixture.databaseID},
			listIDs(t, db, "SELECT database_id FROM backups_restores WHERE id = '"+restoreID.String()+"'"))
		assert.Equal(t, []bool{true}, listBools(t, db,
			"SELECT is_used FROM backups_download_tokens WHERE id = ?", usedTokenID))
		assert.Equal(t, []bool{false}, listBools(t, db,
			"SELECT is_used FROM backups_download_tokens WHERE id = ?", unusedTokenID))
	})

	t.Run("enforces the new constraints after the migration", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, logicalBackupsMigrationName)
		fixture := insertLogicalBackupsFixture(t, db)
		backupID := insertOldLogicalBackup(t, db, fixture, oldLogicalBackup{status: "IN_PROGRESS", encryption: "NONE"})
		insertOldRestore(t, db, backupID, "IN_PROGRESS")
		insertOldDownloadToken(t, db, backupID, fixture.userID, false)
		otherDatabaseID := insertMigratedDatabase(t, db, &fixture.workspaceID)

		applyAllMigrations(t, db, logicalBackupsMigrationName)

		refusedStatements := []struct {
			name      string
			statement string
			values    []any
		}{
			{
				name: "a second in-progress backup of the database",
				statement: `INSERT INTO backups_logical (database_id, storage_id, status, file_name)
					VALUES (?, ?, 'IN_PROGRESS', 'second')`,
				values: []any{fixture.databaseID, fixture.storageID},
			},
			{
				name: "a second in-progress restore of the database",
				statement: `INSERT INTO backups_restores (backup_id, database_id, status)
					VALUES (?, ?, 'IN_PROGRESS')`,
				values: []any{backupID, fixture.databaseID},
			},
			{
				name: "a restore whose database differs from its backup's",
				statement: `INSERT INTO backups_restores (backup_id, database_id, status)
					VALUES (?, ?, 'COMPLETED')`,
				values: []any{backupID, otherDatabaseID},
			},
			{
				name:      "deleting a database that has a backup",
				statement: "DELETE FROM databases WHERE id = ?",
				values:    []any{fixture.databaseID},
			},
			{
				name:      "deleting a user who has a download token",
				statement: "DELETE FROM users WHERE id = ?",
				values:    []any{fixture.userID},
			},
		}

		for _, refusedStatement := range refusedStatements {
			err := db.WithContext(t.Context()).Exec(refusedStatement.statement, refusedStatement.values...).Error
			require.Error(t, err, refusedStatement.name)
		}

		execute(t, db, "DELETE FROM backups_logical WHERE id = ?", backupID)

		assert.Empty(t, listIDs(t, db, "SELECT id FROM backups_restores"))
		assert.Empty(t, listIDs(t, db, "SELECT id FROM backups_download_tokens"))
	})
}

type logicalBackupsFixture struct {
	workspaceID uuid.UUID
	databaseID  uuid.UUID
	storageID   uuid.UUID
	userID      uuid.UUID
}

type oldLogicalBackup struct {
	status             string
	encryption         string
	encryptionSalt     *string
	verificationStatus string
	createdAt          time.Time
}

type statusRow struct {
	Status      string
	FailMessage *string
}

func insertLogicalBackupsFixture(t *testing.T, db *gorm.DB) logicalBackupsFixture {
	t.Helper()

	workspaceID := insertWorkspace(t, db)
	userID := uuid.New()
	execute(t, db, `INSERT INTO users (id, email, password_creation_time, role, name, status)
		VALUES (?, ?, now(), 'MEMBER', 'Owner', 'ACTIVE')`, userID, userID.String()+"@example.com")

	return logicalBackupsFixture{
		workspaceID: workspaceID,
		databaseID:  insertMigratedDatabase(t, db, &workspaceID),
		storageID:   insertStorage(t, db, workspaceID),
		userID:      userID,
	}
}

func insertOldLogicalBackup(
	t *testing.T,
	db *gorm.DB,
	fixture logicalBackupsFixture,
	backup oldLogicalBackup,
) uuid.UUID {
	t.Helper()

	verificationStatus := backup.verificationStatus
	if verificationStatus == "" {
		verificationStatus = "NOT_VERIFIED"
	}

	createdAt := backup.createdAt
	if createdAt.IsZero() {
		createdAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	backupID := uuid.New()
	execute(t, db, `INSERT INTO logical_backups (id, database_id, storage_id, status, encryption, encryption_salt,
		restore_verification_status, file_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		backupID, fixture.databaseID, fixture.storageID, backup.status, backup.encryption, backup.encryptionSalt,
		verificationStatus, "backup-"+backupID.String(), createdAt)

	return backupID
}

func insertOldRestore(t *testing.T, db *gorm.DB, backupID uuid.UUID, status string) uuid.UUID {
	t.Helper()

	restoreID := uuid.New()
	execute(t, db, "INSERT INTO restores (id, backup_id, status) VALUES (?, ?, ?)", restoreID, backupID, status)

	return restoreID
}

func insertOldDownloadToken(t *testing.T, db *gorm.DB, backupID, userID uuid.UUID, isUsed bool) uuid.UUID {
	t.Helper()

	tokenID := uuid.New()
	execute(t, db, `INSERT INTO download_tokens (id, token, backup_id, user_id, expires_at, used)
		VALUES (?, ?, ?, ?, now() + interval '5 minutes', ?)`, tokenID, tokenID.String(), backupID, userID, isUsed)

	return tokenID
}

func readStatusRow(t *testing.T, db *gorm.DB, table string, id uuid.UUID) statusRow {
	t.Helper()

	var row statusRow
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT status, fail_message FROM "+table+" WHERE id = ?", id,
	).Scan(&row).Error)

	return row
}

func listBools(t *testing.T, db *gorm.DB, query string, values ...any) []bool {
	t.Helper()

	var listedValues []bool
	require.NoError(t, db.WithContext(t.Context()).Raw(query, values...).Scan(&listedValues).Error)

	return listedValues
}
