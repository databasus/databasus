package users_testing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
)

func CreateTestUser(ctx context.Context, role users.Role) *users_dto.SignInResponseDTO {
	user := insertTestUser(role, false)

	response, err := users_services.GetUserService().GenerateAccessToken(ctx, user)
	if err != nil {
		panic(err)
	}

	response.Email = user.Email

	return response
}

// An address the profile form would refuse is only reachable this way: the API
// binds the address of every account it creates.
func CreateTestUserWithEmail(
	ctx context.Context,
	role users.Role,
	email string,
) *users_dto.SignInResponseDTO {
	user := insertTestUser(role, false)

	if err := storage.GetDb().WithContext(ctx).Model(&users_models.User{}).
		Where("id = ?", user.ID).
		Update("email", email).Error; err != nil {
		panic(fmt.Errorf("failed to set the address of a test user: %w", err))
	}

	response, err := users_services.GetUserService().GenerateAccessToken(ctx, user)
	if err != nil {
		panic(err)
	}

	response.Email = email

	return response
}

func DeactivateTestUser(ctx context.Context, userID uuid.UUID) {
	userRepository := &users_repositories.UserRepository{}
	if err := userRepository.UpdateUserStatus(userID, users_enums.UserStatusInactive); err != nil {
		panic(fmt.Errorf("failed to deactivate a test user: %w", err))
	}
}

func DeleteTestUser(ctx context.Context, userID uuid.UUID) {
	if err := storage.GetDb().WithContext(ctx).
		Where("id = ?", userID).
		Delete(&users_models.User{}).Error; err != nil {
		panic(fmt.Errorf("failed to delete a test user: %w", err))
	}
}

// The token is minted from the account RecreateInitialAdmin just created rather
// than read back by address, which would hand a nil user to GenerateAccessToken
// once that address changes.
func RecreateInitAdminAndGetAccess(ctx context.Context) *users_dto.SignInResponseDTO {
	user := RecreateInitialAdmin(ctx)

	response, err := users_services.GetUserService().GenerateAccessToken(ctx, user)
	if err != nil {
		panic(err)
	}

	response.Email = user.Email

	return response
}

// The outgoing account is found by the bootstrap-administrator record, not by
// its address: a test that moves the bootstrap administrator to a real address
// would otherwise leave the record behind, and the partial unique index would
// refuse the next bootstrap account.
func RecreateInitialAdmin(ctx context.Context) *users_models.User {
	err := storage.GetDb().WithContext(ctx).Model(&users_models.User{}).
		Where("is_root_admin").
		Updates(map[string]any{
			"email":         "retired-admin-" + uuid.New().String(),
			"is_root_admin": false,
		}).Error
	if err != nil {
		panic(fmt.Errorf("failed to retire the current bootstrap administrator: %w", err))
	}

	return insertTestUser(users.RoleAdmin, true)
}

// No test may assume an instance that holds no account: the tests of one package
// share the database of their test process and carry whatever earlier ones left
// in it, so a test keyed on the account-creation rule has to say so.
func DeleteAllUsers() {
	if err := storage.GetDb().Exec("DELETE FROM users").Error; err != nil {
		panic(fmt.Errorf("failed to empty the users table: %w", err))
	}
}

func insertTestUser(role users.Role, isRootAdmin bool) *users_models.User {
	userID := uuid.New()
	hashedPassword := "$2a$10$test"

	user := &users_models.User{
		ID:                   userID,
		Email:                fmt.Sprintf("%s-%s@test.com", strings.ToLower(string(role)), userID.String()[:8]),
		Name:                 "Test User",
		HashedPassword:       &hashedPassword,
		PasswordCreationTime: time.Now().UTC(),
		CreatedAt:            time.Now().UTC(),
		Role:                 role,
		Status:               users_enums.UserStatusActive,
		IsRootAdmin:          isRootAdmin,
	}

	userRepository := &users_repositories.UserRepository{}
	if err := userRepository.CreateUser(user); err != nil {
		panic(fmt.Errorf("failed to create a test user: %w", err))
	}

	return user
}
