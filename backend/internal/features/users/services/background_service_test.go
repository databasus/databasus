package users_services_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
)

func Test_SweepPendingSignIns_WithRowsOfBothAges_RemovesOnlyThePastItsRetention(t *testing.T) {
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	agedCode := seedPendingSignIn(t, user.UserID, 2*time.Hour)
	freshCode := seedPendingSignIn(t, user.UserID, time.Minute)

	require.NoError(t, users_services.GetUserService().SweepPendingSignInsPastRetention(t.Context()))

	repository := &users_repositories.TwoFactorRepository{}

	sweptCode, err := repository.GetCodeByID(t.Context(), agedCode)
	require.NoError(t, err)
	assert.Nil(t, sweptCode)

	keptCode, err := repository.GetCodeByID(t.Context(), freshCode)
	require.NoError(t, err)
	assert.NotNil(t, keptCode)
}

func Test_SweepExpiredPasswordResetCodes_WithRowsOfBothAges_RemovesOnlyTheExpired(t *testing.T) {
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	expiredCode := seedPasswordResetCode(t, user.UserID, -time.Minute)
	liveCode := seedPasswordResetCode(t, user.UserID, time.Hour)

	require.NoError(t, users_services.GetUserService().SweepExpiredPasswordResetCodes())

	assert.False(t, hasPasswordResetCode(t, expiredCode))
	assert.True(t, hasPasswordResetCode(t, liveCode))
}

func seedPendingSignIn(t *testing.T, userID uuid.UUID, age time.Duration) uuid.UUID {
	hashedCode, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	require.NoError(t, err)

	createdAt := time.Now().UTC().Add(-age)
	code := &users_models.TwoFactorCode{
		ID:                   uuid.New(),
		UserID:               userID,
		HashedCode:           string(hashedCode),
		PasswordCreationTime: createdAt,
		ExpiresAt:            createdAt.Add(10 * time.Minute),
		CreatedAt:            createdAt,
	}

	require.NoError(t, storage.GetDb().WithContext(t.Context()).Create(code).Error)

	return code.ID
}

func seedPasswordResetCode(t *testing.T, userID uuid.UUID, timeToExpiry time.Duration) uuid.UUID {
	hashedCode, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	require.NoError(t, err)

	code := &users_models.PasswordResetCode{
		ID:         uuid.New(),
		UserID:     userID,
		HashedCode: string(hashedCode),
		ExpiresAt:  time.Now().UTC().Add(timeToExpiry),
		CreatedAt:  time.Now().UTC(),
	}

	require.NoError(t, (&users_repositories.PasswordResetRepository{}).CreateResetCode(code))

	return code.ID
}

func hasPasswordResetCode(t *testing.T, codeID uuid.UUID) bool {
	var count int64

	require.NoError(t, storage.GetDb().WithContext(t.Context()).
		Model(&users_models.PasswordResetCode{}).
		Where("id = ?", codeID).
		Count(&count).Error)

	return count > 0
}
