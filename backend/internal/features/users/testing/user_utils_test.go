package users_testing_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_bridge "databasus-backend/internal/features/users"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
)

func Test_CreateTestUser_TokenPassesTheSessionCheckOfTheModule(t *testing.T) {
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	var sessionActor users.Actor
	router := gin.New()
	router.GET("/session", users_testing.GetSessionMiddleware(), func(ctx *gin.Context) {
		sessionActor, _ = users.GetActor(ctx.Request.Context())
		ctx.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/session", nil)
	request.Header.Set("Authorization", "Bearer "+testUser.Token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, users.Actor{UserID: testUser.UserID, Role: users.RoleMember}, sessionActor)
}

func Test_RecreateInitialAdmin_WhenBootstrapAddressHasChanged_MovesRecordToFreshAccount(t *testing.T) {
	outgoingAdmin := users_testing.RecreateInitialAdmin(t.Context())
	changedEmail := "owner-" + uuid.NewString() + "@example.com"
	users_testing.SetTestUserEmail(t.Context(), outgoingAdmin.UserID, changedEmail)

	freshAdmin := users_testing.RecreateInitialAdmin(t.Context())

	var bootstrapAdminIDs []uuid.UUID
	require.NoError(t, storage.GetDb().WithContext(t.Context()).
		Raw("SELECT id FROM users WHERE is_root_admin").Scan(&bootstrapAdminIDs).Error)
	assert.Equal(t, []uuid.UUID{freshAdmin.UserID}, bootstrapAdminIDs)

	retiredAdmin, isFound, err := users_bridge.GetModule().GetUser(t.Context(), outgoingAdmin.UserID)
	require.NoError(t, err)
	require.True(t, isFound)
	assert.NotEqual(t, changedEmail, retiredAdmin.Email)
}

func Test_CreateUser_WhenInstanceAlreadyHasBootstrapAdmin_DatabaseRefusesSecondOne(t *testing.T) {
	users_testing.RecreateInitialAdmin(t.Context())

	err := storage.GetDb().WithContext(t.Context()).Exec(
		`INSERT INTO users (id, email, name, password_creation_time, role, status, is_root_admin, created_at)
		VALUES (?, ?, 'Second Root Admin', ?, 'ADMIN', 'ACTIVE', TRUE, ?)`,
		uuid.New(), "second-root-admin-"+uuid.NewString()+"@example.com", time.Now().UTC(), time.Now().UTC(),
	).Error

	var pgError *pgconn.PgError
	require.True(t, errors.As(err, &pgError), "the insert should fail with a PostgreSQL error: %v", err)
	assert.Equal(t, "idx_users_is_root_admin", pgError.ConstraintName)
}
