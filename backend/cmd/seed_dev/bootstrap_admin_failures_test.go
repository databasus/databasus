package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app"
	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/database"
)

func TestSeedBootstrapAdminFailures(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name    string
		trigger bootstrapTriggerSpec
	}{
		{
			name: "rolls back privileges when the final grant fails",
			trigger: bootstrapTriggerSpec{
				table: "users", condition: "NEW.is_root_admin", action: "RAISE EXCEPTION 'bootstrap grant rejected';",
			},
		},
		{
			name: "rolls back privileges when the transfer audit fails",
			trigger: bootstrapTriggerSpec{
				table: "lib_audit_events", condition: "NEW.message LIKE 'Development seed assigned bootstrap administrator:%'",
				action: "RAISE EXCEPTION 'bootstrap audit rejected';",
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			fixture, params := setUpSeedAccounts(t)
			devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
			previousAdmin := getSeedAccount(t, fixture.router, fixture.admin.UserID)
			devAccount := getSeedAccount(t, fixture.router, devSession.UserId)
			installBootstrapTrigger(t, fixture.router, scenario.trigger)

			exitCode, output := runSeedForTest(t, params)

			require.Equal(t, 1, exitCode, output)
			assert.Contains(t, output, "assign the dev account as bootstrap administrator")
			assert.Equal(t, previousAdmin, getSeedAccount(t, fixture.router, fixture.admin.UserID))
			assert.Equal(t, devAccount, getSeedAccount(t, fixture.router, devSession.UserId))
			assertBootstrapAssignment(t, fixture.router, fixture.admin.UserID)
			assert.Empty(t, listBootstrapEvents(t, fixture.router))
			assert.Empty(t, listSeededItems(t, fixture.router, devSession.Token).workspaceIDs)
		})
	}

	t.Run(
		"preserves the other instance when the database has the same email with another identity",
		func(t *testing.T) {
			t.Parallel()

			fixture, params := setUpSeedAccounts(t)
			devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
			otherApp := apptest.New(t)
			otherSession := signUpForTest(t, otherApp, params.Email, seedPassword)
			otherAccount := getSeedAccount(t, otherApp, otherSession.UserId)
			params.DatabaseDsn = otherApp.GetServerConfig().DatabaseDsn

			exitCode, output := runSeedForTest(t, params)

			require.Equal(t, 1, exitCode, output)
			assert.Contains(t, output, "does not match an active account")
			assert.Equal(t, otherAccount, getSeedAccount(t, otherApp, otherSession.UserId))
			assertBootstrapAssignment(t, fixture.router, fixture.admin.UserID)
			assertBootstrapAssignment(t, otherApp, otherSession.UserId)
			assert.Empty(t, listSeededItems(t, fixture.router, devSession.Token).workspaceIDs)
		},
	)

	t.Run("refuses assignment when the authenticated address no longer matches", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedAccounts(t)
		devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
		request := bootstrapRequest(fixture.router, devSession)
		request.Account.Email = apptest.NewTestEmail("different")

		isChanged, err := app.EnsureDevSeedBootstrapAdmin(t.Context(), request)

		require.ErrorContains(t, err, "does not match an active account")
		assert.False(t, isChanged)
		assertBootstrapAssignment(t, fixture.router, fixture.admin.UserID)
	})

	t.Run("refuses assignment when deactivation completes after authentication", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedAccounts(t)
		devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
		request := bootstrapRequest(fixture.router, devSession)
		deactivated, err := fixture.admin.Client.DeactivateUserWithResponse(t.Context(), devSession.UserId)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, deactivated.StatusCode())

		isChanged, err := app.EnsureDevSeedBootstrapAdmin(t.Context(), request)

		require.ErrorContains(t, err, "does not match an active account")
		assert.False(t, isChanged)
		assertBootstrapAssignment(t, fixture.router, fixture.admin.UserID)
		assert.Equal(t, apiclient.UserStatusINACTIVE, getSeedAccount(t, fixture.router, devSession.UserId).Status)
	})

	for _, scenario := range []struct {
		name        string
		databaseDsn string
	}{
		{name: "reports a safe connection failure when the database settings are invalid", databaseDsn: "postgres://secret-user:secret-password@%invalid/database"},
		{name: "requires the connection string when authentication succeeds", databaseDsn: ""},
		{name: "reports a safe connection failure when the database is unreachable", databaseDsn: "unreachable"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			fixture, params := setUpSeedAccounts(t)
			devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
			params.DatabaseDsn = scenario.databaseDsn

			if params.DatabaseDsn == "unreachable" {
				listenerConfig := net.ListenConfig{}
				listener, err := listenerConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
				require.NoError(t, err)

				_, port, err := net.SplitHostPort(listener.Addr().String())
				require.NoError(t, err)

				require.NoError(t, listener.Close())

				params.DatabaseDsn = fmt.Sprintf(
					"host=127.0.0.1 port=%s user=secret-user password=secret-password dbname=postgres connect_timeout=1",
					port,
				)
			}

			exitCode, output := runSeedForTest(t, params)

			require.Equal(t, 1, exitCode, output)
			assert.Contains(t, output, "assign the dev account as bootstrap administrator")
			assert.NotContains(t, output, "secret-password")
			assert.NotContains(t, output, "secret-user")

			if params.DatabaseDsn != "" {
				assert.NotContains(t, output, params.DatabaseDsn)
			}

			assertBootstrapAssignment(t, fixture.router, fixture.admin.UserID)
			assert.Empty(t, listSeededItems(t, fixture.router, devSession.Token).workspaceIDs)
		})
	}

	t.Run("creates nothing when the seed mode is production", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedAccounts(t)
		params.EnvMode = config.EnvModeProduction

		exitCode, output := runSeedForTest(t, params)

		require.Equal(t, 1, exitCode, output)
		assert.Contains(t, output, "only in development")
		assertBootstrapAssignment(t, fixture.router, fixture.admin.UserID)

		var userCount int
		fixture.router.QuerySQL(t, &userCount, "SELECT COUNT(*) FROM users")

		assert.Equal(t, 1, userCount)

		request := bootstrapRequest(fixture.router, apiclient.Session{})
		request.EnvMode = config.EnvModeProduction

		isChanged, err := app.EnsureDevSeedBootstrapAdmin(t.Context(), request)

		require.ErrorIs(t, err, app.ErrDevSeedNotDevelopment)
		assert.False(t, isChanged)
	})

	t.Run("preserves privileges when registration is closed", func(t *testing.T) {
		t.Parallel()

		router := apptest.New(t)
		admin := router.CreateAdmin(t)
		params := buildTestSeedParams(router.URL)
		params.DatabaseDsn = router.GetServerConfig().DatabaseDsn
		registrations, err := admin.Client.UpdateInstanceSettingsWithResponse(
			t.Context(),
			apiclient.InstanceSettingsUpdate{
				IsAllowExternalRegistrations: false,
			},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, registrations.StatusCode())

		exitCode, output := runSeedForTest(t, params)

		require.Equal(t, 1, exitCode, output)
		assert.Contains(t, output, "cannot register")
		assertBootstrapAssignment(t, router, admin.UserID)
		assert.Empty(t, listBootstrapEvents(t, router))
	})

	t.Run("keeps the completed assignment when workspace creation fails", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedAccounts(t)
		devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
		installBootstrapTrigger(t, fixture.router, bootstrapTriggerSpec{
			table: "workspaces", condition: "TRUE", action: "RAISE EXCEPTION 'workspace creation rejected';",
		})

		exitCode, output := runSeedForTest(t, params)

		require.Equal(t, 1, exitCode, output)
		assert.Contains(t, output, "cannot create workspace")
		assertBootstrapAssignment(t, fixture.router, devSession.UserId)
		assert.Len(t, listBootstrapEvents(t, fixture.router), 1)
		assert.Empty(t, listSeededItems(t, fixture.router, devSession.Token).workspaceIDs)
	})
}

type bootstrapTriggerSpec struct {
	table     string
	condition string
	action    string
}

func installBootstrapTrigger(t *testing.T, router *apptest.App, spec bootstrapTriggerSpec) {
	t.Helper()

	cleanupDB, err := database.Open(t.Context(), router.GetServerConfig().DatabaseDsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		connection, connectionErr := cleanupDB.DB()
		require.NoError(t, connectionErr)
		require.NoError(t, connection.Close())
	})

	router.ExecuteSQL(t, fmt.Sprintf(`CREATE FUNCTION guard_bootstrap_test() RETURNS trigger AS $$ BEGIN
		IF %s THEN %s END IF; RETURN NEW; END $$ LANGUAGE plpgsql`, spec.condition, spec.action))
	router.ExecuteSQL(t, fmt.Sprintf(`CREATE TRIGGER guard_bootstrap_test BEFORE INSERT OR UPDATE ON %s
		FOR EACH ROW EXECUTE FUNCTION guard_bootstrap_test()`, spec.table))

	t.Cleanup(func() {
		cleanup := cleanupDB.WithContext(context.WithoutCancel(t.Context()))
		assert.NoError(t, cleanup.Exec(fmt.Sprintf("DROP TRIGGER guard_bootstrap_test ON %s", spec.table)).Error)
		assert.NoError(t, cleanup.Exec("DROP FUNCTION guard_bootstrap_test()").Error)
	})
}
