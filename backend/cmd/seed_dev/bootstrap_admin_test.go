package main

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app"
	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/users"
)

func TestSeedBootstrapAdmin(t *testing.T) {
	t.Parallel()

	t.Run("assigns bootstrap powers when the dev account is a member", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
		previousAdmin := getSeedAccount(t, fixture.router, fixture.admin.UserID)
		devBefore := getSeedAccount(t, fixture.router, devSession.UserId)

		exitCode, output := runSeedForTest(t, params)

		require.Equal(t, 0, exitCode, output)
		assert.Equal(t, apiclient.UserRoleADMIN,
			getProfile(t, fixture.router, devSession.UserId, devSession.Token).Role)
		assert.Equal(t, apiclient.UserRoleADMIN,
			getProfile(t, fixture.router, fixture.admin.UserID, fixture.admin.Token).Role)

		var isBootstrapAdmin bool
		fixture.router.QuerySQL(t, &isBootstrapAdmin, "SELECT is_root_admin FROM users WHERE id = ?", devSession.UserId)

		assert.True(t, isBootstrapAdmin)
		assertBootstrapAssignment(t, fixture.router, devSession.UserId)

		previousAdmin.IsBootstrapAdmin = false
		assert.Equal(t, previousAdmin, getSeedAccount(t, fixture.router, fixture.admin.UserID))

		devBefore.Role = apiclient.UserRoleADMIN
		devBefore.IsBootstrapAdmin = true
		assert.Equal(t, devBefore, getSeedAccount(t, fixture.router, devSession.UserId))

		member := fixture.router.SignUp(t, apptest.NewTestEmail("member"))

		promoted, err := fixture.router.NewSessionClient(t, devSession.Token).
			ChangeUserRoleWithResponse(t.Context(), member.UserID, apiclient.UserRoleChange{Role: apiclient.UserRoleADMIN})
		require.NoError(t, err)
		require.Equal(t, 200, promoted.StatusCode(), string(promoted.Body))

		demoted, err := fixture.admin.Client.ChangeUserRoleWithResponse(t.Context(), member.UserID,
			apiclient.UserRoleChange{Role: apiclient.UserRoleMEMBER})
		require.NoError(t, err)
		assert.Equal(t, 403, demoted.StatusCode(), string(demoted.Body))

		events := listBootstrapEvents(t, fixture.router)
		require.Len(t, events, 1)
		assert.Contains(t, events[0], fixture.admin.UserID.String())
		assert.Contains(t, events[0], params.Email)
		assert.Contains(t, events[0], devSession.UserId.String())
	})

	t.Run("assigns bootstrap powers when the dev account is a new registration", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)

		exitCode, output := runSeedForTest(t, params)

		require.Equal(t, 0, exitCode, output)

		devSession := signInForTest(t, fixture.router, params.Email, seedPassword)
		assertBootstrapAssignment(t, fixture.router, devSession.UserId)
		assert.Len(t, listBootstrapEvents(t, fixture.router), 1)
	})

	t.Run("keeps the first bootstrap assignment when the instance is empty", func(t *testing.T) {
		t.Parallel()

		router := apptest.New(t)
		params := buildTestSeedParams(router.URL)
		params.DatabaseDsn = router.GetServerConfig().DatabaseDsn
		configureSeedSources(t, &params)

		exitCode, output := runSeedForTest(t, params)

		require.Equal(t, 0, exitCode, output)

		devSession := signInForTest(t, router, params.Email, seedPassword)
		assertBootstrapAssignment(t, router, devSession.UserId)
		assert.Empty(t, listBootstrapEvents(t, router))
	})

	t.Run("preserves bootstrap audit entries when an ordinary admin is seeded twice", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
		promoted, err := fixture.admin.Client.ChangeUserRoleWithResponse(t.Context(), devSession.UserId,
			apiclient.UserRoleChange{Role: apiclient.UserRoleADMIN})
		require.NoError(t, err)
		require.Equal(t, 200, promoted.StatusCode())

		exitCode, output := runSeedForTest(t, params)
		require.Equal(t, 0, exitCode, output)

		accountBefore := getSeedAccount(t, fixture.router, devSession.UserId)
		eventsBefore := listBootstrapEvents(t, fixture.router)
		require.Len(t, eventsBefore, 1)

		exitCode, output = runSeedForTest(t, params)

		require.Equal(t, 0, exitCode, output)
		assert.Equal(t, accountBefore, getSeedAccount(t, fixture.router, devSession.UserId))
		assert.Equal(t, eventsBefore, listBootstrapEvents(t, fixture.router))
	})

	t.Run("records one assignment when two local assignments overlap", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedAccounts(t)
		devSession := signUpForTest(t, fixture.router, params.Email, seedPassword)
		request := bootstrapRequest(fixture.router, devSession)
		assignments := make(chan bool, 2)
		failures := make(chan error, 2)
		var workers sync.WaitGroup

		for range 2 {
			workers.Go(func() {
				isChanged, err := app.EnsureDevSeedBootstrapAdmin(t.Context(), request)
				assignments <- isChanged
				failures <- err
			})
		}
		workers.Wait()

		assert.NoError(t, <-failures)
		assert.NoError(t, <-failures)

		isFirstChanged := <-assignments
		isSecondChanged := <-assignments

		assert.NotEqual(t, isFirstChanged, isSecondChanged)
		assertBootstrapAssignment(t, fixture.router, devSession.UserId)
		assert.Len(t, listBootstrapEvents(t, fixture.router), 1)
	})
}

type seedAccount struct {
	ID               uuid.UUID
	Email            string
	Name             string
	Role             apiclient.UserRole
	Status           apiclient.UserStatus
	HashedPassword   string
	PasswordVersion  uuid.UUID
	IsBootstrapAdmin bool `gorm:"column:is_root_admin"`
}

func getSeedAccount(t *testing.T, router *apptest.App, userID uuid.UUID) seedAccount {
	t.Helper()
	var account seedAccount
	router.QuerySQL(t, &account, "SELECT * FROM users WHERE id = ?", userID)
	return account
}

func assertBootstrapAssignment(t *testing.T, router *apptest.App, userID uuid.UUID) {
	t.Helper()

	var bootstrapIDs []uuid.UUID
	router.QuerySQL(t, &bootstrapIDs, "SELECT id FROM users WHERE is_root_admin")

	assert.Equal(t, []uuid.UUID{userID}, bootstrapIDs)

	account := getSeedAccount(t, router, userID)

	assert.Equal(t, apiclient.UserRoleADMIN, account.Role)
	assert.Equal(t, apiclient.UserStatusACTIVE, account.Status)
}

func listBootstrapEvents(t *testing.T, router *apptest.App) []string {
	t.Helper()

	var messages []string
	router.QuerySQL(
		t,
		&messages,
		"SELECT message FROM lib_audit_events WHERE message LIKE 'Development seed assigned bootstrap administrator:%' ORDER BY id",
	)

	return messages
}

func bootstrapRequest(router *apptest.App, session apiclient.Session) app.DevSeedAdminRequest {
	return app.DevSeedAdminRequest{
		EnvMode:     config.EnvModeDevelopment,
		DatabaseDsn: router.GetServerConfig().DatabaseDsn,
		Account:     users.DevSeedAccount{UserID: session.UserId, Email: session.Email},
		Clock:       router.Clock,
		AuditLogger: logging.NewWithSinks(slog.DiscardHandler).GetAuditLogger(),
	}
}
