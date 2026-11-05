package users_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestGetInstanceSettings(t *testing.T) {
	t.Parallel()

	t.Run("returns the defaults of a new instance with the second factor off", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response, err := admin.Client.GetInstanceSettingsWithResponse(t.Context())

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.True(t, response.JSON200.IsAllowExternalRegistrations)
		assert.True(t, response.JSON200.IsAllowMemberInvitations)
		assert.True(t, response.JSON200.IsMemberAllowedToCreateWorkspaces)
		assert.False(t, response.JSON200.IsTwoFactorAuthRequired)
		assert.True(t, response.JSON200.IsEmailConfigured)
	})

	t.Run("returns the settings to a member", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		member := app.CreateNonAdminUser(t, app.CreateAdmin(t))

		response, err := member.Client.GetInstanceSettingsWithResponse(t.Context())

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("reports no mail server when the operator configured none", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithoutMailServer())
		admin := app.CreateAdmin(t)

		response, err := admin.Client.GetInstanceSettingsWithResponse(t.Context())

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.False(t, response.JSON200.IsEmailConfigured)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		response, err := app.NewClient(t).GetInstanceSettingsWithResponse(t.Context())

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

func TestUpdateInstanceSettings(t *testing.T) {
	t.Parallel()

	t.Run("replaces the settings and reads them back when an admin updates them", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := updateInstanceSettings(t, admin.Client, apiclient.InstanceSettingsUpdate{
			IsAllowExternalRegistrations:      false,
			IsAllowMemberInvitations:          false,
			IsMemberAllowedToCreateWorkspaces: false,
			IsTwoFactorAuthRequired:           true,
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.False(t, response.JSON200.IsAllowExternalRegistrations)
		assert.False(t, response.JSON200.IsAllowMemberInvitations)
		assert.False(t, response.JSON200.IsMemberAllowedToCreateWorkspaces)
		assert.True(t, response.JSON200.IsTwoFactorAuthRequired)

		readSettings, err := admin.Client.GetInstanceSettingsWithResponse(t.Context())
		require.NoError(t, err)
		assert.True(t, readSettings.JSON200.IsTwoFactorAuthRequired)
		assert.False(t, readSettings.JSON200.IsAllowExternalRegistrations)
	})

	t.Run("returns 403 permission_denied and keeps the settings when a member updates them", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		twoFactorUpdate := defaultSettingsUpdate()
		twoFactorUpdate.IsTwoFactorAuthRequired = true

		response := updateInstanceSettings(t, member.Client, twoFactorUpdate)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)

		readSettings, err := admin.Client.GetInstanceSettingsWithResponse(t.Context())
		require.NoError(t, err)
		assert.False(t, readSettings.JSON200.IsTwoFactorAuthRequired)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		response := updateInstanceSettings(t, app.NewClient(t), defaultSettingsUpdate())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 400 when the body is not json", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		request, err := http.NewRequestWithContext(t.Context(), http.MethodPut, app.URL+"/api/v1/instance-settings",
			http.NoBody)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+admin.Token)
		request.Header.Set("Content-Type", "application/json")

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		_ = response.Body.Close()

		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("keeps the mail server answer when the request carries it", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		request, err := http.NewRequestWithContext(t.Context(), http.MethodPut, app.URL+"/api/v1/instance-settings",
			jsonBody(t, `{"isAllowExternalRegistrations":true,"isAllowMemberInvitations":true,`+
				`"isMemberAllowedToCreateWorkspaces":true,"isTwoFactorAuthRequired":false,"isEmailConfigured":false}`))
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+admin.Token)
		request.Header.Set("Content-Type", "application/json")

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		_ = response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)

		readSettings, err := admin.Client.GetInstanceSettingsWithResponse(t.Context())
		require.NoError(t, err)
		assert.True(t, readSettings.JSON200.IsEmailConfigured)
	})

	t.Run("records the old and the new value of each changed switch in the audit log", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		closedRegistration := defaultSettingsUpdate()
		closedRegistration.IsAllowExternalRegistrations = false

		response := updateInstanceSettings(t, admin.Client, closedRegistration)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Contains(t, listGlobalAuditMessages(t, admin), "isAllowExternalRegistrations: true -> false")
	})

	t.Run("records nothing in the audit log when no switch changes", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		auditMessagesBefore := listGlobalAuditMessages(t, admin)

		response := updateInstanceSettings(t, admin.Client, defaultSettingsUpdate())

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Len(t, listGlobalAuditMessages(t, admin), len(auditMessagesBefore))
	})
}

func TestTwoFactorSetting(t *testing.T) {
	t.Parallel()

	t.Run("refuses to turn the second factor on and names the mail server when there is none", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithoutMailServer())
		admin := app.CreateAdmin(t)
		twoFactorUpdate := defaultSettingsUpdate()
		twoFactorUpdate.IsTwoFactorAuthRequired = true

		response := updateInstanceSettings(t, admin.Client, twoFactorUpdate)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeEmailNotConfigured)
		assert.Contains(t, response.ApplicationproblemJSON409.Detail, "mail server")
	})

	t.Run("refuses to turn the second factor on and names the admin whose address is not usable",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			placeholderAdmin := app.CreateNonAdminUser(t, admin)
			app.ExecuteSQL(t, "UPDATE users SET role = 'ADMIN', email = 'admin' WHERE id = ?", placeholderAdmin.UserID)
			twoFactorUpdate := defaultSettingsUpdate()
			twoFactorUpdate.IsTwoFactorAuthRequired = true

			response := updateInstanceSettings(t, admin.Client, twoFactorUpdate)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
				apiclient.ProblemCodeAdminEmailMissing)
			assert.Contains(t, response.ApplicationproblemJSON409.Detail, `"admin"`)

			readSettings, err := admin.Client.GetInstanceSettingsWithResponse(t.Context())
			require.NoError(t, err)
			assert.False(t, readSettings.JSON200.IsTwoFactorAuthRequired)
		})

	t.Run("turns the second factor on when the admin without a usable address is deactivated", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		placeholderAdmin := app.CreateNonAdminUser(t, admin)
		app.ExecuteSQL(t, "UPDATE users SET role = 'ADMIN', email = 'admin', status = 'INACTIVE' WHERE id = ?",
			placeholderAdmin.UserID)
		twoFactorUpdate := defaultSettingsUpdate()
		twoFactorUpdate.IsTwoFactorAuthRequired = true

		response := updateInstanceSettings(t, admin.Client, twoFactorUpdate)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("lets an admin turn the second factor off after the mail server is gone", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithoutMailServer())
		admin := app.CreateAdmin(t)
		requireTwoFactorAfterMailServerLoss(t, app, admin)

		response := updateInstanceSettings(t, admin.Client, defaultSettingsUpdate())

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.False(t, response.JSON200.IsTwoFactorAuthRequired)
	})

	t.Run("keeps a session that predates the switch working", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		twoFactorUpdate := defaultSettingsUpdate()
		twoFactorUpdate.IsTwoFactorAuthRequired = true
		require.Equal(t, http.StatusOK, updateInstanceSettings(t, admin.Client, twoFactorUpdate).StatusCode())

		response := getUser(t, member.Client, member.UserID)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})
}

func TestSignUpWithClosedRegistration(t *testing.T) {
	t.Parallel()

	t.Run("returns 403 registration_closed when the instance has a user and registration is closed",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			closedRegistration := defaultSettingsUpdate()
			closedRegistration.IsAllowExternalRegistrations = false
			require.Equal(t, http.StatusOK, updateInstanceSettings(t, admin.Client, closedRegistration).StatusCode())

			response := signUp(t, app, apptest.NewTestEmail("stranger"), apptest.TestPassword, "Stranger")

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodeRegistrationClosed)
		})

	t.Run("makes the first user the admin when registration is closed on an empty instance", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.ExecuteSQL(
			t,
			"INSERT INTO users_settings (id, is_allow_external_registrations) VALUES (gen_random_uuid(), FALSE)",
		)

		admin := app.SignUp(t, apptest.NewTestEmail("owner"))

		readUser := getUser(t, admin.Client, admin.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserRoleADMIN, readUser.JSON200.Role)
	})
}
