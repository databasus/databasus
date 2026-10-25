package users_testing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
)

func Test_RecreateInitialAdmin_WhenBootstrapAddressHasChanged_MovesRecordToFreshAccount(t *testing.T) {
	repository := &users_repositories.UserRepository{}
	outgoingAdmin := users_testing.RecreateInitialAdmin(t.Context())

	changedEmail := "owner-" + uuid.New().String() + "@example.com"
	require.NoError(t, repository.UpdateUserInfo(outgoingAdmin.ID, nil, &changedEmail))

	freshAdmin := users_testing.RecreateInitialAdmin(t.Context())

	rootAdmin, err := repository.GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	assert.Equal(t, freshAdmin.ID, rootAdmin.ID)

	retiredAdmin, err := repository.GetUserByID(t.Context(), outgoingAdmin.ID)
	require.NoError(t, err)
	assert.False(t, retiredAdmin.IsRootAdmin)
	assert.NotEqual(t, changedEmail, retiredAdmin.Email)
}

func Test_CreateUser_WhenInstanceAlreadyHasBootstrapAdmin_DatabaseRefusesSecondOne(t *testing.T) {
	users_testing.RecreateInitialAdmin(t.Context())

	hashedPassword := "$2a$10$test"
	secondRootAdmin := &users_models.User{
		ID:                   uuid.New(),
		Email:                "second-root-admin-" + uuid.New().String() + "@example.com",
		Name:                 "Second Root Admin",
		HashedPassword:       &hashedPassword,
		PasswordCreationTime: time.Now().UTC(),
		CreatedAt:            time.Now().UTC(),
		Role:                 users.RoleAdmin,
		Status:               users_enums.UserStatusActive,
		IsRootAdmin:          true,
	}

	err := (&users_repositories.UserRepository{}).CreateUser(secondRootAdmin)

	require.Error(t, err)

	var pgError *pgconn.PgError
	require.True(t, errors.As(err, &pgError))
	assert.Equal(t, "idx_users_is_root_admin", pgError.ConstraintName)
}
