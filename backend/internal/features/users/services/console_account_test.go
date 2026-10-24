package users_services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_errors "databasus-backend/internal/features/users/errors"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/storage"
)

const consoleAccountPassword = "console-password-123"

func Test_CreateUserFromConsole_WhenInstanceIsEmpty_CreatesTheBootstrapAdministrator(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()
	recorder := installAuditLogRecorder()
	email := newConsoleEmail()

	outcome, err := createConsoleAccount(t, email, consoleAccountPassword, "admin")
	require.NoError(t, err)

	assert.True(t, outcome.IsCreated)
	assert.True(t, outcome.User.IsRootAdmin)
	assert.Equal(t, users_enums.UserRoleAdmin, outcome.User.Role)
	assert.Contains(t, users_services.RenderConsoleAccountOutcome(outcome), "administrator of this instance")
	assert.True(t, recorder.HasEntryContaining("created from the host console", email, "ADMIN"))
	assert.True(t, recorder.HasEntryContaining("First account on the instance now administers it", email))

	listing, err := users_services.GetManagementService().RenderAdminListing(t.Context())
	require.NoError(t, err)
	assert.Contains(t, adminLineFor(t, listing, email), "bootstrap administrator")

	registered, err := users_services.GetUserService().SignUp(t.Context(), &users_dto.SignUpRequestDTO{
		Email:    newConsoleEmail(),
		Password: "registered-password-123",
		Name:     "Registered",
	})
	require.NoError(t, err)
	assert.Equal(t, users_enums.UserRoleMember, registered.Role)
	assert.False(t, registered.IsRootAdmin)
}

func Test_CreateUserFromConsole_WhenInstanceIsEmptyAndRoleIsMember_RefusesAndCreatesNothing(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()
	installAuditLogRecorder()

	_, err := createConsoleAccount(t, newConsoleEmail(), consoleAccountPassword, "member")

	require.ErrorIs(t, err, users_errors.ErrConsoleAccountFirstMustBeAdmin)

	hasAnyUser, err := users_services.GetUserService().HasAnyUser()
	require.NoError(t, err)
	assert.False(t, hasAnyUser)
}

func Test_CreateUserFromConsole_WhenRegistrationClaimsTheEmptyInstanceConcurrently_LeavesOneBootstrapAdministrator(
	t *testing.T,
) {
	users_testing.ResetSettingsToDefaults(t.Context())
	installAuditLogRecorder()

	for range 5 {
		users_testing.DeleteAllUsers()

		consoleEmail := newConsoleEmail()
		registeredEmail := newConsoleEmail()

		var (
			start        sync.WaitGroup
			done         sync.WaitGroup
			consoleError error
			signUpError  error
		)
		start.Add(1)
		done.Add(2)

		go func() {
			defer done.Done()
			start.Wait()
			_, consoleError = createConsoleAccount(t, consoleEmail, consoleAccountPassword, "admin")
		}()
		go func() {
			defer done.Done()
			start.Wait()
			_, signUpError = users_services.GetUserService().SignUp(t.Context(), &users_dto.SignUpRequestDTO{
				Email:    registeredEmail,
				Password: "registered-password-123",
				Name:     "Registered",
			})
		}()

		start.Done()
		done.Wait()

		require.NoError(t, consoleError)
		require.NoError(t, signUpError)

		assert.Equal(t, int64(1), countBootstrapAdministrators(t))

		consoleUser := getUserByEmail(t, consoleEmail)
		registeredUser := getUserByEmail(t, registeredEmail)
		assert.Equal(t, users_enums.UserRoleAdmin, consoleUser.Role)
		assert.NotEqual(t, consoleUser.IsRootAdmin, registeredUser.IsRootAdmin)
	}
}

func Test_CreateUserFromConsole_WhenInstanceHoldsAccounts_CreatesAnOrdinaryAdministratorThatSignsIn(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	recorder := installAuditLogRecorder()
	email := newConsoleEmail()

	outcome, err := createConsoleAccount(t, email, consoleAccountPassword, "ADMIN")
	require.NoError(t, err)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), outcome.User.ID) })

	assert.True(t, outcome.IsCreated)
	assert.False(t, outcome.User.IsRootAdmin)
	assert.Equal(t, users_enums.UserRoleAdmin, outcome.User.Role)
	assert.NotContains(t, users_services.RenderConsoleAccountOutcome(outcome), "administrator of this instance")
	assert.True(t, recorder.HasEntryContaining("created from the host console", email, "ADMIN"))
	assert.False(t, recorder.HasEntryContaining("First account on the instance"))

	require.NoError(t, signIn(t, email, consoleAccountPassword))
}

func Test_CreateUserFromConsole_WhenExternalRegistrationIsDisabled_CreatesTheMember(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	users_testing.DisableExternalRegistrations(t.Context())
	t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })
	installAuditLogRecorder()
	email := newConsoleEmail()

	outcome, err := createConsoleAccount(t, email, consoleAccountPassword, "member")
	require.NoError(t, err)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), outcome.User.ID) })

	assert.Equal(t, users_enums.UserRoleMember, outcome.User.Role)
	require.NoError(t, signIn(t, email, consoleAccountPassword))
}

func Test_CreateUserFromConsole_WhenPasswordIsResetFromTheConsole_SignsInOnlyWithTheNewPassword(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	installAuditLogRecorder()
	email := newConsoleEmail()

	outcome, err := createConsoleAccount(t, email, consoleAccountPassword, "admin")
	require.NoError(t, err)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), outcome.User.ID) })

	require.NoError(t, users_services.GetUserService().ChangeUserPasswordByEmail(
		t.Context(),
		email,
		"rotated-password-456",
	))

	require.NoError(t, signIn(t, email, "rotated-password-456"))
	require.Error(t, signIn(t, email, consoleAccountPassword))
}

func Test_CreateUserFromConsole_WhenNameIsOmitted_NamesTheAccountAfterTheLocalPart(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	installAuditLogRecorder()
	localPart := "ops-" + uuid.New().String()[:8]

	outcome, err := createConsoleAccount(t, localPart+"@example.com", consoleAccountPassword, "member")
	require.NoError(t, err)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), outcome.User.ID) })

	assert.Equal(t, localPart, outcome.User.Name)
}

func Test_CreateUserFromConsole_WithInvalidInput_RefusesAndCreatesNothing(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	installAuditLogRecorder()

	testCases := []struct {
		name     string
		email    string
		password string
		role     string
		expected error
	}{
		{"not an email address", "ops", consoleAccountPassword, "admin", users_errors.ErrConsoleAccountEmailInvalid},
		{"seven-character password", newConsoleEmail(), "seven77", "admin", users_errors.ErrPasswordTooShort},
		{"missing role", newConsoleEmail(), consoleAccountPassword, "", users_errors.ErrConsoleAccountRoleInvalid},
		{"unknown role", newConsoleEmail(), consoleAccountPassword, "owner", users_errors.ErrConsoleAccountRoleInvalid},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := createConsoleAccount(t, testCase.email, testCase.password, testCase.role)

			require.ErrorIs(t, err, testCase.expected)
			assert.Nil(t, findUserByEmail(t, testCase.email))
		})
	}
}

func Test_CreateUserFromConsole_WhenRunAgainForTheSameAccount_ChangesNothing(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	recorder := installAuditLogRecorder()
	email := newConsoleEmail()

	created, err := createConsoleAccount(t, email, consoleAccountPassword, "admin")
	require.NoError(t, err)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), created.User.ID) })
	entriesAfterCreation := len(recorder.GetEntries())

	rerun, err := createConsoleAccount(t, email, "another-password-456", "admin")
	require.NoError(t, err)

	assert.False(t, rerun.IsCreated)
	assert.Equal(t, created.User.ID, rerun.User.ID)
	assert.Contains(t, users_services.RenderConsoleAccountOutcome(rerun), "password was not changed")
	assert.Len(t, recorder.GetEntries(), entriesAfterCreation)

	require.NoError(t, signIn(t, email, consoleAccountPassword))
	require.Error(t, signIn(t, email, "another-password-456"))
}

func Test_CreateUserFromConsole_WhenAddressBelongsToAnAccountInAnotherState_RefusesAndChangesNothing(
	t *testing.T,
) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	installAuditLogRecorder()

	githubOAuthID := "github-" + uuid.New().String()

	testCases := []struct {
		name          string
		requestedRole string
		existing      *users_models.User
	}{
		{"member asked to become administrator", "admin", newExistingUser(
			users_enums.UserRoleMember, users_enums.UserStatusActive, new("old-hash"), nil,
		)},
		{"deactivated account", "member", newExistingUser(
			users_enums.UserRoleMember, users_enums.UserStatusInactive, new("old-hash"), nil,
		)},
		{"invited account", "member", newExistingUser(
			users_enums.UserRoleMember, users_enums.UserStatusInvited, nil, nil,
		)},
		{"administrator without a password", "admin", newExistingUser(
			users_enums.UserRoleAdmin, users_enums.UserStatusActive, nil, &githubOAuthID,
		)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.NoError(t, (&users_repositories.UserRepository{}).CreateUser(testCase.existing))
			t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), testCase.existing.ID) })

			_, err := createConsoleAccount(t, testCase.existing.Email, consoleAccountPassword, testCase.requestedRole)

			require.ErrorIs(t, err, users_errors.ErrConsoleAccountExistsInAnotherState)

			stored := getUserByEmail(t, testCase.existing.Email)
			assert.Equal(t, testCase.existing.Role, stored.Role)
			assert.Equal(t, testCase.existing.Status, stored.Status)
			assert.Equal(t, testCase.existing.HashedPassword, stored.HashedPassword)
		})
	}
}

func Test_CreateUserFromConsole_InEveryOutcome_EmitsNoPasswordAndNoHash(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	ensureInstanceIsClaimed(t)
	installAuditLogRecorder()
	email := newConsoleEmail()
	rerunPassword := "rerun-password-456"

	created, err := createConsoleAccount(t, email, consoleAccountPassword, "admin")
	require.NoError(t, err)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), created.User.ID) })

	rerun, err := createConsoleAccount(t, email, rerunPassword, "admin")
	require.NoError(t, err)

	_, refusal := createConsoleAccount(t, newConsoleEmail(), rerunPassword, "owner")
	require.Error(t, refusal)

	storedHash := *getUserByEmail(t, email).HashedPassword
	outputs := []string{
		users_services.RenderConsoleAccountOutcome(created),
		users_services.RenderConsoleAccountOutcome(rerun),
		refusal.Error(),
	}

	for _, output := range outputs {
		assert.NotContains(t, output, consoleAccountPassword)
		assert.NotContains(t, output, rerunPassword)
		assert.NotContains(t, output, storedHash)
		assert.NotContains(t, output, "$2a$")
	}
}

func createConsoleAccount(
	t *testing.T,
	email, password, role string,
) (*users_services.ConsoleAccountOutcome, error) {
	t.Helper()

	return users_services.GetUserService().CreateUserFromConsole(
		t.Context(),
		&users_dto.CreateConsoleAccountRequest{Email: email, Password: password, Role: role},
	)
}

// The database of the test process is shared with earlier tests of this package,
// so an instance that holds an account is made rather than assumed.
func ensureInstanceIsClaimed(t *testing.T) {
	t.Helper()

	hasAnyUser, err := users_services.GetUserService().HasAnyUser()
	require.NoError(t, err)

	if !hasAnyUser {
		users_testing.RecreateInitialAdmin(t.Context())
	}
}

func newConsoleEmail() string {
	return "console-" + uuid.New().String() + "@example.com"
}

func newExistingUser(
	role users_enums.UserRole,
	status users_enums.UserStatus,
	hashedPassword *string,
	githubOAuthID *string,
) *users_models.User {
	return &users_models.User{
		ID:                   uuid.New(),
		Email:                newConsoleEmail(),
		Name:                 "Existing",
		HashedPassword:       hashedPassword,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 role,
		Status:               status,
		GitHubOAuthID:        githubOAuthID,
		CreatedAt:            time.Now().UTC(),
	}
}

func findUserByEmail(t *testing.T, email string) *users_models.User {
	t.Helper()

	user, err := users_services.GetUserService().GetUserByEmail(t.Context(), email)
	require.NoError(t, err)

	return user
}

func getUserByEmail(t *testing.T, email string) *users_models.User {
	t.Helper()

	user := findUserByEmail(t, email)
	require.NotNil(t, user)

	return user
}

func countBootstrapAdministrators(t *testing.T) int64 {
	t.Helper()

	var count int64
	require.NoError(t, storage.GetDb().Model(&users_models.User{}).Where("is_root_admin").Count(&count).Error)

	return count
}
