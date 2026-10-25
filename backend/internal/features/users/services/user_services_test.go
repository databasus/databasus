package users_services_test

import (
	"context"
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
	"databasus-backend/internal/modules/users"
)

func Test_ChangeUserPasswordByEmail_WhenNoAccountHoldsTheAddress_ReturnsAnError(t *testing.T) {
	missingEmail := "missing-" + uuid.New().String() + "@example.com"

	err := users_services.GetUserService().ChangeUserPasswordByEmail(
		t.Context(),
		missingEmail,
		"AnyPassword123!",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func Test_ChangeUserPasswordByEmail_WhenPasswordIsShorterThanEightCharacters_KeepsTheOldPassword(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	installAuditLogRecorder()

	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), user.UserID) })

	const oldPassword = "old-password-123"
	require.NoError(t, users_services.GetUserService().ChangeUserPasswordByEmail(
		t.Context(),
		user.Email,
		oldPassword,
	))

	err := users_services.GetUserService().ChangeUserPasswordByEmail(t.Context(), user.Email, "seven77")

	require.ErrorIs(t, err, users_errors.ErrPasswordTooShort)
	require.NoError(t, signIn(t, user.Email, oldPassword))
	require.Error(t, signIn(t, user.Email, "seven77"))
}

func Test_ChangeUserPasswordByEmail_WhenAccountStillHoldsThePlaceholderAddress_SetsThePassword(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()
	installAuditLogRecorder()

	placeholderAdmin := &users_models.User{
		ID:                   uuid.New(),
		Email:                "admin",
		Name:                 "Admin",
		PasswordCreationTime: time.Now().UTC(),
		CreatedAt:            time.Now().UTC(),
		Role:                 users.RoleAdmin,
		Status:               users_enums.UserStatusActive,
		IsRootAdmin:          true,
	}
	require.NoError(t, (&users_repositories.UserRepository{}).CreateUser(placeholderAdmin))
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), placeholderAdmin.ID) })

	require.NoError(t, users_services.GetUserService().ChangeUserPasswordByEmail(
		t.Context(),
		"admin",
		"owner-password-123",
	))

	require.NoError(t, signIn(t, "admin", "owner-password-123"))
}

func installAuditLogRecorder() *users_testing.RecordingAuditLogWriter {
	recorder := users_testing.InstallAuditLogRecorder()
	users_services.GetUserService().SetAuditLogWriter(recorder)

	return recorder
}

func signIn(t *testing.T, email, password string) error {
	t.Helper()

	_, err := users_services.GetUserService().SignIn(
		t.Context(),
		&users_dto.SignInRequestDTO{Email: email, Password: password},
	)

	return err
}
