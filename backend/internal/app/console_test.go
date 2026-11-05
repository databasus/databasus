package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/users"
)

const (
	consolePassword = "console-password-123"
	rotatedPassword = "rotated-password-456"
)

func TestRunConsoleCommand(t *testing.T) {
	t.Run("creates the bootstrap administrator on an instance without users", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		email := newConsoleEmail()

		output, err := createConsoleUser(t, testApp, email, consolePassword, "admin")

		require.NoError(t, err)
		assert.Contains(t, output, "Created account "+email+" with the ADMIN role.")
		assert.Contains(t, output, "It is the administrator of this instance.")
		assert.True(t, hasAuditRecordContaining(t, testApp,
			"Account created from the host console: "+email+" (role "+string(users.RoleAdmin)+")"))
		assert.True(t, hasAuditRecordContaining(t, testApp, "First account on the instance now administers it: "+email))
		assert.Contains(t, getAdminLine(t, testApp, email), "bootstrap administrator")
		require.NoError(t, signIn(t, client, email, consolePassword))

		signedUpEmail := newConsoleEmail()
		signUp(t, client, signedUpEmail)
		signedUpUser, isFound, err := testApp.usersModule.GetUserByEmail(t.Context(), signedUpEmail)
		require.NoError(t, err)
		require.True(t, isFound)
		assert.Equal(t, users.RoleMember, signedUpUser.Role)
		assert.Nil(t, testApp.backgroundGroup)
	})

	t.Run("refuses a member as the first user and creates nothing", func(t *testing.T) {
		testApp := newTestApp(t)

		_, err := createConsoleUser(t, testApp, newConsoleEmail(), consolePassword, "member")

		require.ErrorContains(t, err, "create it with the admin role")
		assert.Zero(t, countUsers(t, testApp))
	})

	t.Run("leaves one bootstrap administrator when a sign-up claims the instance at the same time", func(t *testing.T) {
		for range 5 {
			testApp := newTestApp(t)
			client := newConsoleTestClient(t, testApp)
			consoleEmail := newConsoleEmail()
			signedUpEmail := newConsoleEmail()

			var (
				start        sync.WaitGroup
				done         sync.WaitGroup
				consoleError error
				signUpError  error
				signUpStatus int
			)
			start.Add(1)
			done.Add(2)

			go func() {
				defer done.Done()
				start.Wait()
				_, consoleError = createConsoleUser(t, testApp, consoleEmail, consolePassword, "admin")
			}()
			go func() {
				defer done.Done()
				start.Wait()
				response, err := requestSignUp(t.Context(), client, signedUpEmail)
				signUpError = err
				if err == nil {
					signUpStatus = response.StatusCode()
				}
			}()

			start.Done()
			done.Wait()

			require.NoError(t, consoleError)
			require.NoError(t, signUpError)
			require.Equal(t, http.StatusCreated, signUpStatus)
			assert.Equal(t, 1, countBootstrapAdministrators(t, testApp))
			assert.Contains(t, getAdminLine(t, testApp, consoleEmail), "| active")
		}
	})

	t.Run("creates an ordinary administrator on an instance with users", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		signUp(t, client, newConsoleEmail())
		email := newConsoleEmail()

		output, err := createConsoleUser(t, testApp, email, consolePassword, "ADMIN")

		require.NoError(t, err)
		assert.NotContains(t, output, "administrator of this instance")
		assert.NotContains(t, getAdminLine(t, testApp, email), "bootstrap administrator")
		assert.False(
			t,
			hasAuditRecordContaining(t, testApp, "First account on the instance now administers it: "+email),
		)
		require.NoError(t, signIn(t, client, email, consolePassword))
	})

	t.Run("creates a member while external registration is closed", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		signUp(t, client, newConsoleEmail())
		executeSQL(
			t,
			SQLExecutionSpec{
				App:       testApp,
				Statement: "UPDATE users_settings SET is_allow_external_registrations = FALSE",
			},
		)
		email := newConsoleEmail()

		output, err := createConsoleUser(t, testApp, email, consolePassword, "member")

		require.NoError(t, err)
		assert.Contains(t, output, "with the MEMBER role")
		require.NoError(t, signIn(t, client, email, consolePassword))
	})

	t.Run("names the user after the local part when the name is omitted", func(t *testing.T) {
		testApp := newTestApp(t)
		localPart := "ops-" + uuid.NewString()[:8]
		email := localPart + "@example.com"

		_, err := createConsoleUser(t, testApp, email, consolePassword, "admin")

		require.NoError(t, err)
		user, isFound, err := testApp.usersModule.GetUserByEmail(t.Context(), email)
		require.NoError(t, err)
		require.True(t, isFound)
		assert.Equal(t, localPart, user.Name)
	})

	t.Run("refuses invalid input and creates nothing", func(t *testing.T) {
		testApp := newTestApp(t)

		testCases := []struct {
			name          string
			email         string
			password      string
			role          string
			expectedError string
		}{
			{"not an email address", "ops", consolePassword, "admin", "not a valid email address"},
			{"seven-character password", newConsoleEmail(), "seven77", "admin", "at least 8 characters"},
			{"missing role", newConsoleEmail(), consolePassword, "", "admin or member"},
			{"unknown role", newConsoleEmail(), consolePassword, "owner", "admin or member"},
		}

		for _, testCase := range testCases {
			_, err := createConsoleUser(t, testApp, testCase.email, testCase.password, testCase.role)

			require.ErrorContains(t, err, testCase.expectedError, testCase.name)
		}

		assert.Zero(t, countUsers(t, testApp))
	})

	t.Run("changes nothing when run again for the same user", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		email := createConsoleAdministrator(t, testApp)
		auditRecordCount := countAuditRecords(t, testApp)

		output, err := createConsoleUser(t, testApp, email, "another-password-456", "admin")

		require.NoError(t, err)
		assert.Contains(t, output, "Nothing changed, and its password was not changed.")
		assert.Equal(t, auditRecordCount, countAuditRecords(t, testApp))
		require.NoError(t, signIn(t, client, email, consolePassword))
		require.Error(t, signIn(t, client, email, "another-password-456"))
	})

	t.Run("refuses and changes nothing when the user exists in another state", func(t *testing.T) {
		testApp := newTestApp(t)
		createConsoleAdministrator(t, testApp)

		testCases := []struct {
			name          string
			requestedRole string
			insertedState string
		}{
			{"member asked to become an administrator", "admin", "role = 'MEMBER', status = 'ACTIVE'"},
			{"deactivated user", "member", "role = 'MEMBER', status = 'INACTIVE'"},
			{"invited user", "member", "role = 'MEMBER', status = 'INVITED', hashed_password = NULL"},
			{"administrator without a password", "admin", "role = 'ADMIN', status = 'ACTIVE', hashed_password = NULL"},
		}

		for _, testCase := range testCases {
			email := newConsoleEmail()
			_, err := createConsoleUser(t, testApp, email, consolePassword, "member")
			require.NoError(t, err)
			executeSQL(t, SQLExecutionSpec{
				App:       testApp,
				Statement: "UPDATE users SET " + testCase.insertedState + " WHERE email = ?",
				Values:    []any{email},
			})
			storedStateBefore := readStoredState(t, testApp, email)

			_, err = createConsoleUser(t, testApp, email, rotatedPassword, testCase.requestedRole)

			require.ErrorContains(t, err, "already exists", testCase.name)
			assert.Equal(t, storedStateBefore, readStoredState(t, testApp, email), testCase.name)
		}
	})

	t.Run("prints neither the password nor its hash in any outcome", func(t *testing.T) {
		testApp := newTestApp(t)
		email := newConsoleEmail()

		createdOutput, err := createConsoleUser(t, testApp, email, consolePassword, "admin")
		require.NoError(t, err)
		rerunOutput, err := createConsoleUser(t, testApp, email, rotatedPassword, "admin")
		require.NoError(t, err)
		_, refusal := createConsoleUser(t, testApp, newConsoleEmail(), rotatedPassword, "owner")
		require.Error(t, refusal)

		storedHash := readStoredState(t, testApp, email).HashedPassword
		for _, output := range []string{createdOutput, rerunOutput, refusal.Error()} {
			assert.NotContains(t, output, consolePassword)
			assert.NotContains(t, output, rotatedPassword)
			assert.NotContains(t, output, storedHash)
			assert.NotContains(t, output, "$2a$")
		}
	})

	t.Run("resets the password so only the new one signs in and records the change", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		email := createConsoleAdministrator(t, testApp)

		_, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
			Kind: ConsoleCommandResetPassword, Email: email, NewPassword: rotatedPassword,
		})

		require.NoError(t, err)
		require.NoError(t, signIn(t, client, email, rotatedPassword))
		require.Error(t, signIn(t, client, email, consolePassword))
		assert.Equal(t, int64(1), countPasswordChangeRecords(t, testApp, email))
		assert.Nil(t, testApp.backgroundGroup)
	})

	t.Run("refuses a reset to a password shorter than 8 characters", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		email := createConsoleAdministrator(t, testApp)

		_, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
			Kind: ConsoleCommandResetPassword, Email: email, NewPassword: "seven77",
		})

		require.ErrorContains(t, err, "at least 8 characters")
		require.NoError(t, signIn(t, client, email, consolePassword))
	})

	t.Run("sets the password of the user that still holds the placeholder address", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		email := createConsoleAdministrator(t, testApp)
		executeSQL(t, SQLExecutionSpec{
			App: testApp, Statement: "UPDATE users SET email = 'admin' WHERE email = ?", Values: []any{email},
		})

		_, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
			Kind: ConsoleCommandResetPassword, Email: "admin", NewPassword: rotatedPassword,
		})

		require.NoError(t, err)
		require.NoError(t, signIn(t, client, "admin", rotatedPassword))
	})

	t.Run("returns an error and changes no user when the address is unknown", func(t *testing.T) {
		testApp := newTestApp(t)
		client := newConsoleTestClient(t, testApp)
		email := createConsoleAdministrator(t, testApp)

		_, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
			Kind: ConsoleCommandResetPassword, Email: newConsoleEmail(), NewPassword: rotatedPassword,
		})

		require.Error(t, err)
		require.NoError(t, signIn(t, client, email, consolePassword))
	})

	t.Run("switches the second factor off and records it in the audit log", func(t *testing.T) {
		testApp := newTestApp(t)
		executeSQL(
			t,
			SQLExecutionSpec{
				App:       testApp,
				Statement: "INSERT INTO users_settings (is_two_factor_auth_required) VALUES (TRUE)",
			},
		)

		output, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{Kind: ConsoleCommandDisableTwoFactor})

		require.NoError(t, err)
		assert.Contains(t, output, "Two-factor authentication is now off")
		assert.False(t, isTwoFactorAuthRequired(t, testApp))
		assert.True(
			t,
			hasAuditRecordContaining(
				t,
				testApp,
				"isTwoFactorAuthRequired: true -> false (switched off from the host console)",
			),
		)
		assert.Nil(t, testApp.backgroundGroup)
	})

	t.Run("reports no change when the second factor is already off", func(t *testing.T) {
		testApp := newTestApp(t)

		output, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{Kind: ConsoleCommandDisableTwoFactor})

		require.NoError(t, err)
		assert.Contains(t, output, "Two-factor authentication was already off")
		assert.False(t, hasAuditRecordContaining(t, testApp, "isTwoFactorAuthRequired"))
	})

	t.Run("lists the administrators unmasked, marks the bootstrap one and leaves members out", func(t *testing.T) {
		testApp := newTestApp(t)
		bootstrapEmail := createConsoleAdministrator(t, testApp)
		ordinaryEmail := createConsoleAdministrator(t, testApp)
		deactivatedEmail := createConsoleAdministrator(t, testApp)
		executeSQL(t, SQLExecutionSpec{
			App:       testApp,
			Statement: "UPDATE users SET status = 'INACTIVE' WHERE email = ?",
			Values:    []any{deactivatedEmail},
		})
		memberEmail := newConsoleEmail()
		_, err := createConsoleUser(t, testApp, memberEmail, consolePassword, "member")
		require.NoError(t, err)

		output, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{Kind: ConsoleCommandListAdmins})

		require.NoError(t, err)
		assert.NotContains(t, output, memberEmail)
		assert.NotContains(t, output, "$2a$")
		assert.Contains(t, getLineFor(t, output, bootstrapEmail), "| active | bootstrap administrator")
		assert.NotContains(t, getLineFor(t, output, ordinaryEmail), "bootstrap administrator")
		assert.Contains(t, getLineFor(t, output, ordinaryEmail), "| active")
		assert.Contains(t, getLineFor(t, output, deactivatedEmail), "| inactive")
		assert.Nil(t, testApp.backgroundGroup)
	})

	t.Run("lists no administrator on an instance without users", func(t *testing.T) {
		testApp := newTestApp(t)

		output, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{Kind: ConsoleCommandListAdmins})

		require.NoError(t, err)
		assert.Equal(t, "This instance has no administrator: no account has been created on it yet.", output)
	})
}

func createConsoleAdministrator(t *testing.T, testApp *App) string {
	t.Helper()

	email := newConsoleEmail()

	_, err := createConsoleUser(t, testApp, email, consolePassword, "admin")
	require.NoError(t, err)

	return email
}

func createConsoleUser(t *testing.T, testApp *App, email, password, role string) (string, error) {
	t.Helper()

	return testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
		Kind: ConsoleCommandCreateUser, Email: email, Password: password, Role: role,
	})
}

func newConsoleEmail() string {
	return "console-" + strings.ReplaceAll(uuid.NewString(), "-", "") + "@databasus.local"
}

// newConsoleTestClient serves the application's handler, so a test signs in the way a user does.
func newConsoleTestClient(t *testing.T, testApp *App) *apiclient.ClientWithResponses {
	t.Helper()

	handler, err := testApp.NewHandler()
	require.NoError(t, err)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return apiclient.New(t, server.URL)
}

// signIn returns an error unless the password sign-in issues a session.
func signIn(t *testing.T, client *apiclient.ClientWithResponses, email, password string) error {
	t.Helper()

	response, err := client.SignInWithResponse(t.Context(), apiclient.SignInRequest{
		Email:                    email,
		Password:                 &password,
		CloudflareTurnstileToken: nil,
	})
	require.NoError(t, err)

	if response.StatusCode() != http.StatusOK {
		return errors.New(string(response.Body))
	}

	return nil
}

// requestSignUp asserts nothing, so a goroutine other than the test's may call it.
func requestSignUp(
	ctx context.Context,
	client *apiclient.ClientWithResponses,
	email string,
) (*apiclient.SignUpResponse, error) {
	password := consolePassword

	return client.SignUpWithResponse(ctx, apiclient.SignUpRequest{
		Email:                    openapi_types.Email(email),
		Password:                 &password,
		Name:                     "Signed Up",
		CloudflareTurnstileToken: nil,
	})
}

func signUp(t *testing.T, client *apiclient.ClientWithResponses, email string) {
	t.Helper()

	response, err := requestSignUp(t.Context(), client, email)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
}

func getAdminLine(t *testing.T, testApp *App, email string) string {
	t.Helper()

	listing, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{Kind: ConsoleCommandListAdmins})
	require.NoError(t, err)

	return getLineFor(t, listing, email)
}

func getLineFor(t *testing.T, listing, email string) string {
	t.Helper()

	for line := range strings.SplitSeq(listing, "\n") {
		if strings.HasPrefix(line, email+" |") {
			return line
		}
	}

	t.Fatalf("no line for %s in the listing:\n%s", email, listing)

	return ""
}

type storedUserState struct {
	Role           string
	Status         string
	HashedPassword string
}

func readStoredState(t *testing.T, testApp *App, email string) storedUserState {
	t.Helper()

	var state storedUserState
	require.NoError(t, testApp.db.WithContext(t.Context()).Raw(
		"SELECT role, status, COALESCE(hashed_password, '') AS hashed_password FROM users WHERE email = ?", email,
	).Scan(&state).Error)

	return state
}

type SQLExecutionSpec struct {
	App       *App
	Statement string
	Values    []any
}

func executeSQL(t *testing.T, spec SQLExecutionSpec) {
	t.Helper()

	require.NoError(t, spec.App.db.WithContext(t.Context()).Exec(spec.Statement, spec.Values...).Error)
}

func countUsers(t *testing.T, testApp *App) int {
	t.Helper()

	var userCount int
	require.NoError(
		t,
		testApp.db.WithContext(t.Context()).Raw("SELECT COUNT(*) FROM users").Scan(&userCount).Error,
	)

	return userCount
}

func countBootstrapAdministrators(t *testing.T, testApp *App) int {
	t.Helper()

	var bootstrapAdminCount int
	require.NoError(t, testApp.db.WithContext(t.Context()).Raw(
		"SELECT COUNT(*) FROM users WHERE is_root_admin",
	).Scan(&bootstrapAdminCount).Error)

	return bootstrapAdminCount
}

func countAuditRecords(t *testing.T, testApp *App) int {
	t.Helper()

	var recordCount int
	require.NoError(t, testApp.db.WithContext(t.Context()).Raw(
		"SELECT COUNT(*) FROM lib_audit_events",
	).Scan(&recordCount).Error)

	return recordCount
}

func isTwoFactorAuthRequired(t *testing.T, testApp *App) bool {
	t.Helper()

	var isRequired bool
	require.NoError(t, testApp.db.WithContext(t.Context()).Raw(
		"SELECT is_two_factor_auth_required FROM users_settings",
	).Scan(&isRequired).Error)

	return isRequired
}

func hasAuditRecordContaining(t *testing.T, testApp *App, messagePart string) bool {
	t.Helper()

	var isPresent bool
	require.NoError(t, testApp.db.WithContext(t.Context()).Raw(
		"SELECT EXISTS (SELECT 1 FROM lib_audit_events WHERE strpos(message, ?) > 0)", messagePart,
	).Scan(&isPresent).Error)

	return isPresent
}

func countPasswordChangeRecords(t *testing.T, testApp *App, email string) int64 {
	t.Helper()

	var recordCount int64
	require.NoError(t, testApp.db.WithContext(t.Context()).Raw(
		`SELECT COUNT(*) FROM lib_audit_events
		 WHERE message = 'Password changed' AND author_email = ?`,
		email,
	).Scan(&recordCount).Error)

	return recordCount
}
