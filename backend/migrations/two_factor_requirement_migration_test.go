package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/migrations"
)

const (
	twoFactorRequirementMigrationName    = "20261124120000_users_authenticator_apps.sql"
	twoFactorRequirementMigrationVersion = 20261124120000
)

func TestTwoFactorRequirementMigration(t *testing.T) {
	t.Parallel()

	t.Run("maps the required boolean to EMAIL and the optional one to OFF", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, twoFactorRequirementMigrationName)
		requiredSettingsID := insertTwoFactorSettings(t, db, true)
		optionalSettingsID := insertTwoFactorSettings(t, db, false)

		applyAllMigrations(t, db, twoFactorRequirementMigrationName)

		assert.Equal(t, "EMAIL", getTwoFactorRequirement(t, db, requiredSettingsID))
		assert.Equal(t, "OFF", getTwoFactorRequirement(t, db, optionalSettingsID))
	})

	t.Run("rolls back to the boolean and drops the pending sign-ins without an emailed code", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, twoFactorRequirementMigrationName)
		applyAllMigrations(t, db, twoFactorRequirementMigrationName)
		emailSettingsID := uuid.New()
		appSettingsID := uuid.New()
		offSettingsID := uuid.New()
		for settingsID, requirement := range map[uuid.UUID]string{
			emailSettingsID: "EMAIL",
			appSettingsID:   "AUTHENTICATOR_APP",
			offSettingsID:   "OFF",
		} {
			execute(
				t,
				db,
				"INSERT INTO users_settings (id, two_factor_requirement) VALUES (?, ?)",
				settingsID,
				requirement,
			)
		}
		userID := uuid.New()
		execute(t, db, `INSERT INTO users (id, email, name, password_creation_time, role, status)
			VALUES (?, 'app-user@example.com', 'App user', NOW(), 'MEMBER', 'ACTIVE')`, userID)
		appID := uuid.New()
		execute(t, db, `INSERT INTO users_authenticator_apps (id, user_id, encrypted_secret, connected_at)
			VALUES (?, ?, 'enc:secret', NOW())`, appID, userID)
		emailPendingSignInID := uuid.New()
		execute(t, db, `INSERT INTO users_pending_sign_ins
			(id, user_id, kind, hashed_code, password_version, expires_at, created_at)
			VALUES (?, ?, 'EMAIL_CODE', 'hashed-code', gen_random_uuid(), NOW() + interval '10 minutes', NOW())`,
			emailPendingSignInID, userID)
		execute(t, db, `INSERT INTO users_pending_sign_ins
			(id, user_id, kind, authenticator_app_id, password_version, expires_at, created_at)
			VALUES (gen_random_uuid(), ?, 'AUTHENTICATOR_CODE', ?, gen_random_uuid(), NOW() + interval '10 minutes', NOW())`,
			userID, appID)

		rollBackTwoFactorRequirementMigration(t, db)

		assert.Equal(t, []uuid.UUID{emailPendingSignInID}, listIDs(t, db, "SELECT id FROM users_pending_sign_ins"))
		assert.True(t, getIsTwoFactorAuthRequired(t, db, emailSettingsID))
		assert.True(t, getIsTwoFactorAuthRequired(t, db, appSettingsID))
		assert.False(t, getIsTwoFactorAuthRequired(t, db, offSettingsID))
	})
}

func insertTwoFactorSettings(t *testing.T, db *gorm.DB, isRequired bool) uuid.UUID {
	t.Helper()

	settingsID := uuid.New()
	execute(t, db, "INSERT INTO users_settings (id, is_two_factor_auth_required) VALUES (?, ?)", settingsID, isRequired)

	return settingsID
}

func getTwoFactorRequirement(t *testing.T, db *gorm.DB, settingsID uuid.UUID) string {
	t.Helper()

	requirements := listStrings(t, db, "SELECT two_factor_requirement FROM users_settings WHERE id = ?", settingsID)
	require.Len(t, requirements, 1)

	return requirements[0]
}

func getIsTwoFactorAuthRequired(t *testing.T, db *gorm.DB, settingsID uuid.UUID) bool {
	t.Helper()

	var isRequired bool
	require.NoError(t, db.WithContext(t.Context()).
		Raw("SELECT is_two_factor_auth_required FROM users_settings WHERE id = ?", settingsID).
		Scan(&isRequired).Error)

	return isRequired
}

func rollBackTwoFactorRequirementMigration(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		migrations.Files,
		goose.WithDisableGlobalRegistry(true),
	)
	require.NoError(t, err)

	_, err = provider.ApplyVersion(t.Context(), twoFactorRequirementMigrationVersion, false)
	require.NoError(t, err)
}
