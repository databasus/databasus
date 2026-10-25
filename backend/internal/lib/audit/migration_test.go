package audit_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/migrations"
)

const auditEventsMigrationName = "20261025095000_move_audit_logs_to_lib_audit_events.sql"

type legacyAuditRow struct {
	id          uuid.UUID
	userID      *uuid.UUID
	workspaceID *uuid.UUID
	message     string
	createdAt   time.Time
}

func TestAuditEventsMigration(t *testing.T) {
	t.Parallel()

	t.Run("keeps every event and snapshots the authors that still exist", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBeforeAuditEvents(t)
		alice := insertLegacyUser(t, db, "alice@example.com", "Alice")
		bob := insertLegacyUser(t, db, "bob@example.com", "Bob")
		workspaceID := uuid.New()
		legacyRows := []legacyAuditRow{
			newLegacyAuditRow(&alice, nil, "Storage created: nightly", time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)),
			newLegacyAuditRow(&bob, &workspaceID, "Database deleted: payments",
				time.Date(2026, 4, 2, 8, 30, 15, 123456000, time.UTC)),
			newLegacyAuditRow(nil, &workspaceID, "Backup failed", time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)),
		}
		insertLegacyAuditRows(t, db, legacyRows)

		applyAllMigrations(t, db)

		storedEventsByID := listStoredEventsByID(t, db)
		require.Len(t, storedEventsByID, len(legacyRows))
		for _, legacyRow := range legacyRows {
			storedEvent := storedEventsByID[legacyRow.id]
			assert.Equal(t, legacyRow.message, storedEvent.Message)
			assert.Equal(t, legacyRow.workspaceID, storedEvent.WorkspaceID)
			assert.True(t, legacyRow.createdAt.Equal(storedEvent.CreatedAt),
				"%s became %s", legacyRow.createdAt, storedEvent.CreatedAt)
		}

		assertAuthor(t, storedEventsByID[legacyRows[0].id], alice, "alice@example.com", "Alice")
		assertAuthor(t, storedEventsByID[legacyRows[1].id], bob, "bob@example.com", "Bob")
		assertNoAuthor(t, storedEventsByID[legacyRows[2].id])
	})

	t.Run("leaves an event without an author when its account no longer exists", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBeforeAuditEvents(t)
		require.NoError(t, db.WithContext(t.Context()).Exec(
			"ALTER TABLE audit_logs DROP CONSTRAINT fk_audit_logs_user_id",
		).Error)
		removedUserID := uuid.New()
		orphanedRow := newLegacyAuditRow(&removedUserID, nil, "Storage created: nightly",
			time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
		insertLegacyAuditRows(t, db, []legacyAuditRow{orphanedRow})

		applyAllMigrations(t, db)

		storedEventsByID := listStoredEventsByID(t, db)
		require.Len(t, storedEventsByID, 1)
		assert.Equal(t, orphanedRow.message, storedEventsByID[orphanedRow.id].Message)
		assertNoAuthor(t, storedEventsByID[orphanedRow.id])
	})
}

func openDatabaseMigratedBeforeAuditEvents(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), testdb.OpenEmpty(t))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	migrationFiles, err := fs.ReadDir(migrations.Files, ".")
	require.NoError(t, err)

	earlierMigrations := fstest.MapFS{}
	for _, migrationFile := range migrationFiles {
		if !strings.HasSuffix(migrationFile.Name(), ".sql") || migrationFile.Name() >= auditEventsMigrationName {
			continue
		}

		content, err := fs.ReadFile(migrations.Files, migrationFile.Name())
		require.NoError(t, err)

		earlierMigrations[migrationFile.Name()] = &fstest.MapFile{Data: content}
	}

	_, err = database.Migrate(t.Context(), db, earlierMigrations)
	require.NoError(t, err)

	return db
}

func applyAllMigrations(t *testing.T, db *gorm.DB) {
	t.Helper()

	appliedNames, err := database.Migrate(t.Context(), db, migrations.Files)
	require.NoError(t, err)
	require.Contains(t, appliedNames, auditEventsMigrationName)
}

func insertLegacyUser(t *testing.T, db *gorm.DB, email, name string) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	require.NoError(t, db.WithContext(t.Context()).Exec(
		`INSERT INTO users (id, email, hashed_password, password_creation_time, created_at, role, name, status)
		VALUES (?, ?, NULL, NOW(), NOW(), 'MEMBER', ?, 'ACTIVE')`,
		userID, email, name,
	).Error)

	return userID
}

func newLegacyAuditRow(
	userID, workspaceID *uuid.UUID,
	message string,
	createdAt time.Time,
) legacyAuditRow {
	return legacyAuditRow{
		id:          uuid.New(),
		userID:      userID,
		workspaceID: workspaceID,
		message:     message,
		createdAt:   createdAt,
	}
}

// insertLegacyAuditRows writes created_at as the wall-clock text the old code stored in its
// TIMESTAMP column, so the session time zone cannot shift it.
func insertLegacyAuditRows(t *testing.T, db *gorm.DB, legacyRows []legacyAuditRow) {
	t.Helper()

	for _, legacyRow := range legacyRows {
		require.NoError(t, db.WithContext(t.Context()).Exec(
			`INSERT INTO audit_logs (id, user_id, workspace_id, message, created_at)
			VALUES (?, ?, ?, ?, CAST(? AS TIMESTAMP))`,
			legacyRow.id, legacyRow.userID, legacyRow.workspaceID, legacyRow.message,
			legacyRow.createdAt.Format("2006-01-02 15:04:05.999999"),
		).Error)
	}
}

func listStoredEventsByID(t *testing.T, db *gorm.DB) map[uuid.UUID]storedEvent {
	t.Helper()

	storedEventsByID := map[uuid.UUID]storedEvent{}
	for _, event := range listStoredEvents(t, db) {
		storedEventsByID[event.ID] = event
	}

	return storedEventsByID
}

func assertAuthor(t *testing.T, event storedEvent, userID uuid.UUID, email, name string) {
	t.Helper()

	assert.Equal(t, &userID, event.AuthorUserID)
	assert.Equal(t, &email, event.AuthorEmail)
	assert.Equal(t, &name, event.AuthorName)
}

func assertNoAuthor(t *testing.T, event storedEvent) {
	t.Helper()

	assert.Nil(t, event.AuthorUserID)
	assert.Nil(t, event.AuthorEmail)
	assert.Nil(t, event.AuthorName)
}
