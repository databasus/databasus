package users_controllers

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/email"
	users_dto "databasus-backend/internal/features/users/dto"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_AdminLifecycleE2E_CompletesSuccessfully(t *testing.T) {
	router := createE2ETestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	ownerEmail := "owner" + uuid.New().String() + "@example.com"
	signUpRequest := users_dto.SignUpRequestDTO{
		Email:    ownerEmail,
		Password: "ownerpassword123",
		Name:     "Owner",
	}

	var signUpResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signup",
		"",
		signUpRequest,
		http.StatusOK,
		&signUpResponse,
	)

	var profileResponse users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+signUpResponse.Token,
		http.StatusOK,
		&profileResponse,
	)
	assert.Equal(t, users.RoleAdmin, profileResponse.Role)

	rootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	assert.Equal(t, signUpResponse.UserID, rootAdmin.ID)

	var signInResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: ownerEmail, Password: "ownerpassword123"},
		http.StatusOK,
		&signInResponse,
	)

	changedEmail := "changed" + uuid.New().String() + "@example.com"
	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+signInResponse.Token,
		users_dto.UpdateUserInfoRequestDTO{Email: &changedEmail},
		http.StatusOK,
	)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: changedEmail, Password: "ownerpassword123"},
		http.StatusOK,
	)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: ownerEmail, Password: "ownerpassword123"},
		http.StatusBadRequest,
	)

	rootAdminAfterChange, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdminAfterChange)
	assert.Equal(t, signUpResponse.UserID, rootAdminAfterChange.ID)
	assert.Equal(t, changedEmail, rootAdminAfterChange.Email)
}

func Test_UserLifecycleE2E_CompletesSuccessfully(t *testing.T) {
	router := createE2ETestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())

	// An account has to exist first: the registration below is an ordinary one
	// only because the instance has already been claimed.
	users_testing.RecreateInitialAdmin(t.Context())

	// 1. User registers
	userEmail := "testuser" + uuid.New().String() + "@example.com"
	userSignupRequest := users_dto.SignUpRequestDTO{
		Email:    userEmail,
		Password: "userpassword123",
		Name:     "Test User",
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		userSignupRequest,
		http.StatusOK,
	)

	// 2. User signs in
	userSigninRequest := users_dto.SignInRequestDTO{
		Email:    userEmail,
		Password: "userpassword123",
	}

	var signinResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		userSigninRequest,
		http.StatusOK,
		&signinResponse,
	)
	assert.NotEmpty(t, signinResponse.Token)
	assert.NotEqual(t, uuid.Nil, signinResponse.UserID)

	// 3. User gets own profile
	var profileResponse users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/"+signinResponse.UserID.String(),
		"Bearer "+signinResponse.Token,
		http.StatusOK,
		&profileResponse,
	)
	assert.Equal(t, signinResponse.UserID, profileResponse.ID)
	assert.Equal(t, userEmail, profileResponse.Email)
	assert.Equal(t, users.RoleMember, profileResponse.Role)
	assert.True(t, profileResponse.IsActive)
}

// Test router creation helpers
func createUserTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register public routes
	GetUserController().RegisterRoutes(v1)

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetUserController().RegisterProtectedRoutes(protected.(*gin.RouterGroup))

	// Setup audit log service
	auditLogRecorder := users_testing.InstallAuditLogRecorder()
	users_services.GetUserService().SetAuditLogWriter(auditLogRecorder)

	return router
}

func createSettingsTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetSettingsController().RegisterRoutes(protected.(*gin.RouterGroup))

	// Setup audit log service
	auditLogRecorder := users_testing.InstallAuditLogRecorder()
	users_services.GetUserService().SetAuditLogWriter(auditLogRecorder)
	users_services.GetSettingsService().SetAuditLogWriter(auditLogRecorder)
	users_services.GetManagementService().SetAuditLogWriter(auditLogRecorder)

	return router
}

// A settings test that reads the mail-server answer has to own the sender the
// answer comes from, because the wired one reports whatever the environment
// running the suite happens to configure.
func createSettingsTestRouterWithMailSender(
	t *testing.T,
	isMailServerMissing bool,
) (*gin.Engine, *users_testing.MockEmailSender) {
	router := createSettingsTestRouter()

	mockEmailSender := users_testing.NewMockEmailSender()
	mockEmailSender.IsMailServerMissing = isMailServerMissing
	users_services.GetSettingsService().SetEmailSender(mockEmailSender)

	// The services are process-global, so a mock left installed would decide
	// what every later test in the package believes about the mail server.
	t.Cleanup(func() {
		users_services.GetSettingsService().SetEmailSender(email.GetInstanceSender())
		users_testing.ResetSettingsToDefaults(context.Background())
	})

	return router, mockEmailSender
}

func createManagementTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetManagementController().RegisterRoutes(protected.(*gin.RouterGroup))

	// Setup audit log service
	auditLogRecorder := users_testing.InstallAuditLogRecorder()
	users_services.GetUserService().SetAuditLogWriter(auditLogRecorder)
	users_services.GetSettingsService().SetAuditLogWriter(auditLogRecorder)
	users_services.GetManagementService().SetAuditLogWriter(auditLogRecorder)

	return router
}

func createE2ETestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register all routes
	GetUserController().RegisterRoutes(v1)

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetUserController().RegisterProtectedRoutes(protected.(*gin.RouterGroup))
	GetSettingsController().RegisterRoutes(protected.(*gin.RouterGroup))
	GetManagementController().RegisterRoutes(protected.(*gin.RouterGroup))

	// Setup audit log service
	auditLogRecorder := users_testing.InstallAuditLogRecorder()
	users_services.GetUserService().SetAuditLogWriter(auditLogRecorder)
	users_services.GetSettingsService().SetAuditLogWriter(auditLogRecorder)
	users_services.GetManagementService().SetAuditLogWriter(auditLogRecorder)

	return router
}
