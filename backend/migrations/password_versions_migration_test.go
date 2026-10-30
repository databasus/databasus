package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestPasswordVersionsMigration(t *testing.T) {
	t.Parallel()

	t.Run("assigns distinct versions and preserves unusable pending history after upgrade", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261030210000_password_versions.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		for _, email := range []string{"one@example.com", "two@example.com"} {
			execute(
				t,
				db,
				"INSERT INTO users (email, name, password_creation_time, role, status) VALUES (?, 'User', NOW(), 'MEMBER', 'ACTIVE')",
				email,
			)
		}
		execute(t, db, `INSERT INTO users_pending_sign_ins
   (user_id, hashed_code, password_creation_time, expires_at, failed_attempt_count)
   SELECT id, 'hashed-code', password_creation_time, NOW() + interval '10 minutes', 2 FROM users`)

		applyAllMigrations(t, db, migrationName)

		userVersions := listIDs(t, db, "SELECT password_version FROM users")
		require.Len(t, userVersions, 2)
		assert.NotEqual(t, userVersions[0], userVersions[1])
		pendingVersions := listIDs(t, db, "SELECT password_version FROM users_pending_sign_ins")
		require.Len(t, pendingVersions, 2)
		assert.NotEqual(t, pendingVersions[0], pendingVersions[1])
		for _, version := range append(userVersions, pendingVersions...) {
			assert.NotEqual(t, uuid.Nil, version)
		}
		for _, version := range pendingVersions {
			assert.NotContains(t, userVersions, version)
		}
		var retainedCount int
		require.NoError(
			t,
			db.Raw("SELECT count(*) FROM users_pending_sign_ins WHERE failed_attempt_count = 2 AND NOT is_used").
				Scan(&retainedCount).
				Error,
		)
		assert.Equal(t, 2, retainedCount)
	})

	t.Run("gives each new user a version on a fresh installation", func(t *testing.T) {
		t.Parallel()
		db, err := database.Open(t.Context(), testdb.Open(t))
		require.NoError(t, err)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { _ = sqlDB.Close() })
		execute(
			t,
			db,
			"INSERT INTO users (email, name, password_creation_time, role, status) VALUES ('fresh-one@example.com', 'One', NOW(), 'MEMBER', 'ACTIVE'), ('fresh-two@example.com', 'Two', NOW(), 'MEMBER', 'ACTIVE')",
		)
		versions := listIDs(t, db, "SELECT password_version FROM users")
		require.Len(t, versions, 2)
		assert.NotEqual(t, versions[0], versions[1])
		assert.NotEqual(t, uuid.Nil, versions[0])
		assert.NotEqual(t, uuid.Nil, versions[1])
	})
}
