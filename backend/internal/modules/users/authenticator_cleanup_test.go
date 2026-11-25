package users_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/modules/users"
)

func TestAuthenticatorCleanup(t *testing.T) {
	t.Parallel()

	t.Run("deletes expired setups and wrong codes older than an hour and keeps younger ones", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		requireStartedSetup(t, startProfileSetup(t, member, nil))
		regenerateRecoveryCodes(t, admin, new(getWrongCode(app.GenerateAuthenticatorCode(t, connectedApp.Secret))))
		app.Clock.Advance(50 * time.Minute)
		youngSetup := requireStartedSetup(t, startProfileSetup(t, admin,
			new(app.GenerateAuthenticatorCode(t, connectedApp.Secret))))
		regenerateRecoveryCodes(t, admin, new(getWrongCode(app.GenerateAuthenticatorCode(t, connectedApp.Secret))))
		app.Clock.Advance(15 * time.Minute)

		app.RunJobPass(t, "users_expired_codes_cleanup")

		var remainingSetupIDs []string
		app.QuerySQL(t, &remainingSetupIDs, "SELECT id::text FROM users_authenticator_setups")
		assert.Equal(t, []string{youngSetup.AuthenticatorAppSetupId.String()}, remainingSetupIDs)
		assert.Equal(t, 1, countWrongCodes(t, app, admin.UserID))
	})
}

func TestRecoveryNormalization(t *testing.T) {
	t.Parallel()

	t.Run("drops setups and wrong codes and keeps connected apps with their recovery codes", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		connectedApp := app.ConnectAuthenticatorApp(t, admin)
		requireStartedSetup(t, startProfileSetup(t, member, nil))
		regenerateRecoveryCodes(t, admin, new(getWrongCode(app.GenerateAuthenticatorCode(t, connectedApp.Secret))))

		app.RunOnDatabase(t, users.NormalizeRecoveredState)

		var setupCount, appCount, recoveryCodeCount int
		app.QuerySQL(t, &setupCount, "SELECT count(*) FROM users_authenticator_setups")
		app.QuerySQL(t, &appCount, "SELECT count(*) FROM users_authenticator_apps")
		app.QuerySQL(t, &recoveryCodeCount, "SELECT count(*) FROM users_recovery_codes")
		assert.Zero(t, setupCount)
		assert.Zero(t, countWrongCodes(t, app, admin.UserID))
		assert.Equal(t, 1, appCount)
		assert.Equal(t, 10, recoveryCodeCount)
	})
}
