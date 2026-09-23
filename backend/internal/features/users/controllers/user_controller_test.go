package users_controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/util/ratelimiter"
	test_utils "databasus-backend/internal/util/testing"
)

type failingRateLimitCounter struct {
	err error
}

const bootstrapAdminPassword = "ownerpassword123"

func (c failingRateLimitCounter) RecordAttemptAndCheckIsAllowed(
	context.Context,
	ratelimiter.Attempt,
) (bool, error) {
	return false, c.err
}

func Test_SignIn_WhenRateLimiterFails_RejectsWithoutLoggingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logOutput bytes.Buffer
	controller := &UserController{
		userService: nil,
		rateLimiter: failingRateLimitCounter{err: errors.New("counter failed")},
		logger:      slog.New(slog.NewTextHandler(&logOutput, nil)),
	}
	router := gin.New()
	router.POST("/users/signin", controller.SignIn)

	email := "private@example.com"
	request := httptest.NewRequest(
		http.MethodPost,
		"/users/signin",
		strings.NewReader(`{"email":"`+email+`","password":"password"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.NotContains(t, logOutput.String(), email)
}

func Test_SignUpUser_WithValidData_UserCreated(t *testing.T) {
	router := createUserTestRouter()

	request := users_dto.SignUpRequestDTO{
		Email:    "test" + uuid.New().String() + "@example.com",
		Password: "testpassword123",
		Name:     "Test User",
	}

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signup",
		"",
		request,
		http.StatusOK,
		&response,
	)

	assert.NotEmpty(t, response.Token)
	assert.NotEqual(t, uuid.Nil, response.UserID)
	assert.Equal(t, request.Email, response.Email)
}

func Test_SignUpUser_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/signup",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_SignUpUser_WithDuplicateEmail_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	email := "duplicate" + uuid.New().String() + "@example.com"

	request := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "testpassword123",
		Name:     "Test User",
	}

	// First signup
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", request, http.StatusOK)

	// Second signup with same email
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "already exists")
}

func Test_SignUpUser_WithValidationErrors_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	testCases := []struct {
		name    string
		request users_dto.SignUpRequestDTO
	}{
		{
			name: "missing email",
			request: users_dto.SignUpRequestDTO{
				Password: "testpassword123",
				Name:     "Test User",
			},
		},
		{
			name: "missing password",
			request: users_dto.SignUpRequestDTO{
				Email: "test@example.com",
				Name:  "Test User",
			},
		},
		{
			name: "short password",
			request: users_dto.SignUpRequestDTO{
				Email:    "test@example.com",
				Password: "short",
				Name:     "Test User",
			},
		},
		{
			name: "missing name",
			request: users_dto.SignUpRequestDTO{
				Email:    "test@example.com",
				Password: "testpassword123",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/users/signup",
				"",
				tc.request,
				http.StatusBadRequest,
			)
		})
	}
}

func Test_SignInUser_WithValidCredentials_ReturnsToken(t *testing.T) {
	router := createUserTestRouter()
	email := "signin" + uuid.New().String() + "@example.com"
	password := "testpassword123"

	// First create a user
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: password,
		Name:     "Test User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Now sign in
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: password,
	}

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signinRequest,
		http.StatusOK,
		&response,
	)

	assert.NotEmpty(t, response.Token)
	assert.NotEqual(t, uuid.Nil, response.UserID)
}

func Test_SignInUser_WithWrongPassword_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	email := "signin2" + uuid.New().String() + "@example.com"

	// First create a user
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "testpassword123",
		Name:     "Test User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Now sign in with wrong password
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: "wrongpassword",
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signinRequest,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "password is incorrect")
}

func Test_SignInUser_WithNonExistentUser_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	signinRequest := users_dto.SignInRequestDTO{
		Email:    "nonexistent" + uuid.New().String() + "@example.com",
		Password: "testpassword123",
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signinRequest,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "does not exist")
}

func Test_SignInUser_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/signin",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_ChangeUserPassword_WithValidData_PasswordChanged(t *testing.T) {
	router := createUserTestRouter()
	email := "changepass" + uuid.New().String() + "@example.com"
	oldPassword := "oldpassword123"
	newPassword := "newpassword123"

	// Create user via signup
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: oldPassword,
		Name:     "Test User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Sign in to get token
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: oldPassword,
	}
	var signinResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signinRequest,
		http.StatusOK,
		&signinResponse,
	)

	// Change password
	changePasswordRequest := users_dto.ChangePasswordRequestDTO{
		NewPassword: newPassword,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/change-password",
		"Bearer "+signinResponse.Token,
		changePasswordRequest,
		http.StatusOK,
	)

	// Verify old password no longer works
	oldSigninRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: oldPassword,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		oldSigninRequest,
		http.StatusBadRequest,
	)

	// Verify new password works
	newSigninRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: newPassword,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		newSigninRequest,
		http.StatusOK,
	)
}

func Test_ChangeUserPassword_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	router := createUserTestRouter()

	request := users_dto.ChangePasswordRequestDTO{
		NewPassword: "newpassword123",
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/change-password",
		"",
		request,
		http.StatusUnauthorized,
	)
}

func Test_ChangeUserPassword_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	testUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/change-password",
		Body:           "invalid json",
		AuthToken:      "Bearer " + testUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_ChangeUserPassword_WithValidationErrors_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	testUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	testCases := []struct {
		name    string
		request users_dto.ChangePasswordRequestDTO
	}{
		{
			name:    "missing new password",
			request: users_dto.ChangePasswordRequestDTO{},
		},
		{
			name: "short new password",
			request: users_dto.ChangePasswordRequestDTO{
				NewPassword: "short",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test_utils.MakePutRequest(
				t,
				router,
				"/api/v1/users/change-password",
				"Bearer "+testUser.Token,
				tc.request,
				http.StatusBadRequest,
			)
		})
	}
}

func Test_InviteUser_WhenUserIsAdmin_UserInvited(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	workspaceID := uuid.New()
	workspaceRole := users_enums.WorkspaceRoleMember

	request := users_dto.InviteUserRequestDTO{
		Email:                 "invited" + uuid.New().String() + "@example.com",
		IntendedWorkspaceID:   &workspaceID,
		IntendedWorkspaceRole: &workspaceRole,
	}

	var response users_dto.InviteUserResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminUser.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, request.Email, response.Email)
	assert.Equal(t, request.IntendedWorkspaceID, response.IntendedWorkspaceID)
	assert.Equal(t, request.IntendedWorkspaceRole, response.IntendedWorkspaceRole)
	assert.NotEqual(t, uuid.Nil, response.ID)
}

func Test_InviteUser_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	router := createUserTestRouter()

	request := users_dto.InviteUserRequestDTO{
		Email: "invited@example.com",
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"",
		request,
		http.StatusUnauthorized,
	)
}

func Test_InviteUser_WithoutPermission_ReturnsForbidden(t *testing.T) {
	router := createUserTestRouter()
	defer users_testing.ResetSettingsToDefaults(t.Context())

	memberUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	uniqueID := uuid.New().String()[:8]
	request := users_dto.InviteUserRequestDTO{
		Email: fmt.Sprintf("invited_%s@example.com", uniqueID),
	}

	users_testing.DisableMemberInvitations(t.Context())

	settingsService := users_services.GetSettingsService()
	settings, err := settingsService.GetSettings(t.Context())
	assert.NoError(t, err)

	if settings.IsAllowMemberInvitations {
		t.Fatal(
			"RACE CONDITION DETECTED: Member invitations should be disabled but were enabled by another test",
		)
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+memberUser.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")
}

func Test_InviteUser_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/invite",
		Body:           "invalid json",
		AuthToken:      "Bearer " + adminUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_InviteUser_WithValidationErrors_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)

	testCases := []struct {
		name    string
		request users_dto.InviteUserRequestDTO
	}{
		{
			name: "missing email",
			request: users_dto.InviteUserRequestDTO{
				IntendedWorkspaceID: &uuid.UUID{},
			},
		},
		{
			name: "invalid email",
			request: users_dto.InviteUserRequestDTO{
				Email: "invalid-email",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/users/invite",
				"Bearer "+adminUser.Token,
				tc.request,
				http.StatusBadRequest,
			)
		})
	}
}

func Test_InviteUser_WithDuplicateEmail_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	email := "duplicate-invite" + uuid.New().String() + "@example.com"

	request := users_dto.InviteUserRequestDTO{
		Email: email,
	}

	// First invitation
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminUser.Token,
		request,
		http.StatusOK,
	)

	// Second invitation with same email
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminUser.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "already exists")
}

func Test_UpdateUserInfo_WithValidName_NameUpdated(t *testing.T) {
	router := createUserTestRouter()
	testUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	newName := "Updated Name"
	request := users_dto.UpdateUserInfoRequestDTO{
		Name: &newName,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+testUser.Token,
		request,
		http.StatusOK,
	)

	var profile users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&profile,
	)

	assert.Equal(t, "Updated Name", profile.Name)
}

func Test_UpdateUserInfo_WithValidEmail_EmailUpdated(t *testing.T) {
	router := createUserTestRouter()
	testUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	newEmail := "newemail" + uuid.New().String() + "@example.com"
	request := users_dto.UpdateUserInfoRequestDTO{
		Email: &newEmail,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+testUser.Token,
		request,
		http.StatusOK,
	)

	var profile users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&profile,
	)

	assert.Equal(t, newEmail, profile.Email)
}

func Test_UpdateUserInfo_WithTakenEmail_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	user1 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	request := users_dto.UpdateUserInfoRequestDTO{
		Email: &user2.Email,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+user1.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "already taken")
}

func Test_GitHubOAuth_WithValidCode_ReturnsToken(t *testing.T) {
	testID := uuid.New().String()[:8]
	testEmail := "github-user-" + testID + "@example.com"
	testOAuthID := int64(uuid.New().ID())

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    testOAuthID,
				"email": testEmail,
				"name":  "GitHub Test User",
				"login": "githubtest",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/login/oauth/authorize",
		TokenURL: mockServer.URL + "/login/oauth/access_token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/user",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, testEmail, response.Email)
	assert.True(t, response.IsNewUser)
}

func Test_GitHubOAuth_WithExistingEmail_LinksAccount(t *testing.T) {
	testID := uuid.New().String()[:8]
	email := "existing-" + testID + "@example.com"
	testOAuthID := int64(uuid.New().ID())

	router := createUserTestRouter()
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "testpassword123",
		Name:     "Existing User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    testOAuthID,
				"email": email,
				"name":  "GitHub Test User",
				"login": "githubtest",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/login/oauth/authorize",
		TokenURL: mockServer.URL + "/login/oauth/access_token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/user",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, email, response.Email)
	assert.False(t, response.IsNewUser)
}

func Test_GitHubOAuth_WithInvitedUser_ActivatesUser(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	testID := uuid.New().String()[:8]
	email := "invited-" + testID + "@example.com"
	testOAuthID := int64(uuid.New().ID())

	inviteRequest := users_dto.InviteUserRequestDTO{
		Email: email,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminUser.Token,
		inviteRequest,
		http.StatusOK,
	)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    testOAuthID,
				"email": email,
				"name":  "GitHub Test User",
				"login": "githubtest",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/login/oauth/authorize",
		TokenURL: mockServer.URL + "/login/oauth/access_token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/user",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, email, response.Email)
	assert.False(t, response.IsNewUser)
}

func Test_GitHubOAuth_WithNoPublicEmail_FetchesFromEmailsEndpoint(t *testing.T) {
	testID := uuid.New().String()[:8]
	testEmail := "private-email-" + testID + "@example.com"
	testOAuthID := int64(uuid.New().ID())

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    testOAuthID,
				"email": "",
				"name":  "GitHub Test User",
				"login": "githubtest",
			})
			return
		}
		if r.URL.Path == "/user/emails" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"email":    testEmail,
					"primary":  true,
					"verified": true,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/login/oauth/authorize",
		TokenURL: mockServer.URL + "/login/oauth/access_token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/user",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, testEmail, response.Email)
	assert.True(t, response.IsNewUser)
}

func Test_GitHubOAuth_WhenRegistrationDisabled_ReturnsBadRequest(t *testing.T) {
	defer users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DisableExternalRegistrations(t.Context())

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    99999,
				"email": "new-user-" + uuid.New().String()[:8] + "@example.com",
				"name":  "GitHub Test User",
				"login": "githubtest",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/login/oauth/authorize",
		TokenURL: mockServer.URL + "/login/oauth/access_token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/user",
	)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "registration is disabled")
}

func Test_GoogleOAuth_WithValidCode_ReturnsToken(t *testing.T) {
	testID := uuid.New().String()[:8]
	testEmail := "google-user-" + testID + "@example.com"
	testOAuthID := "google-" + testID

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
			})
			return
		}
		if r.URL.Path == "/userinfo" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    testOAuthID,
				"email": testEmail,
				"name":  "Google Test User",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/auth",
		TokenURL: mockServer.URL + "/token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGoogleOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/userinfo",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, testEmail, response.Email)
	assert.True(t, response.IsNewUser)
}

func Test_GoogleOAuth_WithExistingEmail_LinksAccount(t *testing.T) {
	testID := uuid.New().String()[:8]
	email := "existing-google-" + testID + "@example.com"
	testOAuthID := "google-" + testID + "-456"

	router := createUserTestRouter()
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "testpassword123",
		Name:     "Existing User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
			})
			return
		}
		if r.URL.Path == "/userinfo" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    testOAuthID,
				"email": email,
				"name":  "Google Test User",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/auth",
		TokenURL: mockServer.URL + "/token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGoogleOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/userinfo",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, email, response.Email)
	assert.False(t, response.IsNewUser)
}

func Test_GoogleOAuth_WithInvitedUser_ActivatesUser(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	testID := uuid.New().String()[:8]
	email := "invited-google-" + testID + "@example.com"
	testOAuthID := "google-" + testID + "-789"

	inviteRequest := users_dto.InviteUserRequestDTO{
		Email: email,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminUser.Token,
		inviteRequest,
		http.StatusOK,
	)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
			})
			return
		}
		if r.URL.Path == "/userinfo" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    testOAuthID,
				"email": email,
				"name":  "Google Test User",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	endpoint := oauth2.Endpoint{
		AuthURL:  mockServer.URL + "/auth",
		TokenURL: mockServer.URL + "/token",
	}

	userService := users_services.GetUserService()
	response, err := userService.HandleGoogleOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		endpoint,
		mockServer.URL+"/userinfo",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, email, response.Email)
	assert.False(t, response.IsNewUser)
}

func Test_SignIn_WithExcessiveAttempts_RateLimitEnforced(t *testing.T) {
	router := createUserTestRouter()
	email := "ratelimit" + uuid.New().String() + "@example.com"
	password := "testpassword123"

	// Create a user first
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: password,
		Name:     "Rate Limit Test User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Make 10 sign-in attempts (should succeed)
	for range 10 {
		signinRequest := users_dto.SignInRequestDTO{
			Email:    email,
			Password: password,
		}
		test_utils.MakePostRequest(
			t,
			router,
			"/api/v1/users/signin",
			"",
			signinRequest,
			http.StatusOK,
		)
	}

	// 11th attempt should be rate limited
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: password,
	}
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signinRequest,
		http.StatusTooManyRequests,
	)
	assert.Contains(t, string(resp.Body), "Rate limit exceeded")
}

func Test_SignUp_WhenInstanceHoldsNoAccount_AccountAdministersTheInstance(t *testing.T) {
	router := createUserTestRouter()
	auditLogRecorder := users_testing.GetAuditLogRecorder()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	email := "first" + uuid.New().String() + "@example.com"
	signUpResponse := signUpViaAPI(t, router, email)

	profile := getOwnProfile(t, router, signUpResponse.Token)
	assert.Equal(t, users_enums.UserRoleAdmin, profile.Role)

	rootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	assert.Equal(t, signUpResponse.UserID, rootAdmin.ID)

	assert.True(t, auditLogRecorder.HasEntryContaining("administers", email))
}

func Test_SignUp_WhenInstanceAlreadyHoldsAnAccount_AccountIsOrdinaryMember(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	signUpViaAPI(t, router, "first"+uuid.New().String()+"@example.com")
	secondResponse := signUpViaAPI(t, router, "second"+uuid.New().String()+"@example.com")

	profile := getOwnProfile(t, router, secondResponse.Token)
	assert.Equal(t, users_enums.UserRoleMember, profile.Role)

	rootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	assert.NotEqual(t, secondResponse.UserID, rootAdmin.ID)
}

func Test_SignUp_WhenInstanceIsEmptyAndExternalRegistrationDisabled_AccountStillTakesTheInstance(t *testing.T) {
	router := createUserTestRouter()
	defer users_testing.ResetSettingsToDefaults(t.Context())

	users_testing.DisableExternalRegistrations(t.Context())
	users_testing.DeleteAllUsers()

	signUpResponse := signUpViaAPI(t, router, "first"+uuid.New().String()+"@example.com")

	profile := getOwnProfile(t, router, signUpResponse.Token)
	assert.Equal(t, users_enums.UserRoleAdmin, profile.Role)
}

func Test_SignUp_WhenInstanceHoldsAnAccountAndExternalRegistrationDisabled_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	defer users_testing.ResetSettingsToDefaults(t.Context())

	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.RecreateInitialAdmin(t.Context())
	users_testing.DisableExternalRegistrations(t.Context())

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		users_dto.SignUpRequestDTO{
			Email:    "refused" + uuid.New().String() + "@example.com",
			Password: "userpassword123",
			Name:     "Refused User",
		},
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "external registration is disabled")
}

func Test_SignUp_WhenBootstrapAdminAlreadyRecorded_RegistersAsMemberAndLeavesRecordAlone(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())

	bootstrapAdmin := users_testing.RecreateInitialAdmin(t.Context())

	signUpResponse := signUpViaAPI(t, router, "member"+uuid.New().String()+"@example.com")

	profile := getOwnProfile(t, router, signUpResponse.Token)
	assert.Equal(t, users_enums.UserRoleMember, profile.Role)

	rootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	assert.Equal(t, bootstrapAdmin.ID, rootAdmin.ID)
}

func Test_SignUp_WhenEmailBelongsToAnotherAccount_ReturnsDuplicateRefusal(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.RecreateInitialAdmin(t.Context())

	takenEmail := "taken" + uuid.New().String() + "@example.com"
	signUpViaAPI(t, router, takenEmail)

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		users_dto.SignUpRequestDTO{
			Email:    takenEmail,
			Password: "userpassword123",
			Name:     "Duplicate User",
		},
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "already exists")
}

func Test_SignUp_WhenEmailIsNotAnAddress_ReturnsBadRequestAndCreatesNoAccount(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		users_dto.SignUpRequestDTO{
			Email:    "admin",
			Password: "userpassword123",
			Name:     "Not An Address",
		},
		http.StatusBadRequest,
	)

	assert.False(t, hasAnyUserViaAPI(t, router))

	signUpViaAPI(t, router, "valid"+uuid.New().String()+"@example.com")
	assert.True(t, hasAnyUserViaAPI(t, router))
}

func Test_HasAnyUser_WhenInstanceHoldsNoAccount_AnswersFalse(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	assert.False(t, hasAnyUserViaAPI(t, router))
}

func Test_HasAnyUser_WhenInstanceHoldsAnAccount_AnswersTrue(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()
	users_testing.RecreateInitialAdmin(t.Context())

	assert.True(t, hasAnyUserViaAPI(t, router))
}

func Test_HasAnyUser_WhenInstanceHoldsAnAccountAndExternalRegistrationEnabled_AllowsSignUp(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.RecreateInitialAdmin(t.Context())

	assert.True(t, getEntryStateViaAPI(t, router).IsSignUpAllowed)
}

func Test_HasAnyUser_WhenInstanceHoldsAnAccountAndExternalRegistrationDisabled_RefusesSignUp(t *testing.T) {
	router := createUserTestRouter()
	defer users_testing.ResetSettingsToDefaults(t.Context())

	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.RecreateInitialAdmin(t.Context())
	users_testing.DisableExternalRegistrations(t.Context())

	assert.False(t, getEntryStateViaAPI(t, router).IsSignUpAllowed)
}

func Test_HasAnyUser_WhenInstanceIsEmptyAndExternalRegistrationDisabled_StillAllowsSignUp(t *testing.T) {
	router := createUserTestRouter()
	defer users_testing.ResetSettingsToDefaults(t.Context())

	users_testing.DisableExternalRegistrations(t.Context())
	users_testing.DeleteAllUsers()

	assert.True(t, getEntryStateViaAPI(t, router).IsSignUpAllowed)
}

// The spec refuses any way to set a password on an existing account without
// authenticating as it or proving control of its address.
func Test_SetPasswordOnExistingAccount_WhenCallerIsAnonymous_IsRefused(t *testing.T) {
	router := createUserTestRouter()

	test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         http.MethodGet,
		URL:            "/api/v1/users/admin/has-password",
		ExpectedStatus: http.StatusNotFound,
	})

	test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         http.MethodPost,
		URL:            "/api/v1/users/admin/set-password",
		Body:           `{"password":"adminpassword123"}`,
		ExpectedStatus: http.StatusNotFound,
	})
}

func Test_GitHubOAuth_WhenInstanceHoldsNoAccount_AccountAdministersTheInstance(t *testing.T) {
	createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	email := "github-first-" + uuid.New().String()[:8] + "@example.com"
	mockServer := newGitHubOAuthMockServer(email, int64(uuid.New().ID()))
	defer mockServer.Close()

	response, err := users_services.GetUserService().HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		gitHubOAuthEndpoint(mockServer.URL),
		mockServer.URL+"/user",
	)

	require.NoError(t, err)
	assertAccountAdministersTheInstance(t, response.UserID)
}

func Test_GoogleOAuth_WhenInstanceHoldsNoAccount_AccountAdministersTheInstance(t *testing.T) {
	createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	email := "google-first-" + uuid.New().String()[:8] + "@example.com"
	mockServer := newGoogleOAuthMockServer(email, "google-"+uuid.New().String()[:8])
	defer mockServer.Close()

	response, err := users_services.GetUserService().HandleGoogleOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		googleOAuthEndpoint(mockServer.URL),
		mockServer.URL+"/userinfo",
	)

	require.NoError(t, err)
	assertAccountAdministersTheInstance(t, response.UserID)
}

func assertAccountAdministersTheInstance(t *testing.T, userID uuid.UUID) {
	t.Helper()

	repository := &users_repositories.UserRepository{}

	user, err := repository.GetUserByID(t.Context(), userID)
	require.NoError(t, err)
	assert.Equal(t, users_enums.UserRoleAdmin, user.Role)

	rootAdmin, err := repository.GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	assert.Equal(t, userID, rootAdmin.ID)
}

func signUpViaAPI(t *testing.T, router *gin.Engine, email string) users_dto.SignInResponseDTO {
	t.Helper()

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signup",
		"",
		users_dto.SignUpRequestDTO{
			Email:    email,
			Password: "userpassword123",
			Name:     "Test User",
		},
		http.StatusOK,
		&response,
	)

	return response
}

func getOwnProfile(t *testing.T, router *gin.Engine, token string) users_dto.UserProfileResponseDTO {
	t.Helper()

	var profile users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+token,
		http.StatusOK,
		&profile,
	)

	return profile
}

func hasAnyUserViaAPI(t *testing.T, router *gin.Engine) bool {
	t.Helper()

	return getEntryStateViaAPI(t, router).HasAnyUser
}

func getEntryStateViaAPI(t *testing.T, router *gin.Engine) users_dto.HasAnyUserResponseDTO {
	t.Helper()

	var entryState users_dto.HasAnyUserResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/is-any-user-exist",
		"",
		http.StatusOK,
		&entryState,
	)

	return entryState
}

func gitHubOAuthEndpoint(serverURL string) oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  serverURL + "/login/oauth/authorize",
		TokenURL: serverURL + "/login/oauth/access_token",
	}
}

func googleOAuthEndpoint(serverURL string) oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  serverURL + "/auth",
		TokenURL: serverURL + "/token",
	}
}

func newGitHubOAuthMockServer(email string, oauthID int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "bearer",
			})
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    oauthID,
				"email": email,
				"name":  "GitHub Test User",
				"login": "githubtest",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newGoogleOAuthMockServer(email, oauthID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
			})
		case "/userinfo":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    oauthID,
				"email": email,
				"name":  "Google Test User",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func Test_UpdateUserInfo_WhenBootstrapAdminChangesEmail_SignInMovesToTheNewAddress(t *testing.T) {
	router := createUserTestRouter()
	auditLogRecorder := users_testing.GetAuditLogRecorder()
	owner, ownerEmail := claimInstanceViaAPI(t, router)

	changedEmail := "changed" + uuid.New().String() + "@example.com"
	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+owner.Token,
		users_dto.UpdateUserInfoRequestDTO{Email: &changedEmail},
		http.StatusOK,
	)

	signInViaAPI(t, router, changedEmail, http.StatusOK)
	signInViaAPI(t, router, ownerEmail, http.StatusBadRequest)

	rootAdmin, err := (&users_repositories.UserRepository{}).GetRootAdmin(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rootAdmin)
	assert.Equal(t, owner.UserID, rootAdmin.ID)

	assert.True(t, auditLogRecorder.HasEntryContaining("Email changed from", ownerEmail, changedEmail))

	// The token carries the user id and the password creation time, never the
	// address, so it survives the change.
	profile := getOwnProfile(t, router, owner.Token)
	assert.Equal(t, changedEmail, profile.Email)
}

func Test_UpdateUserInfo_WhenBootstrapAdminSubmitsUnusableEmail_ChangeIsRejected(t *testing.T) {
	router := createUserTestRouter()
	owner, ownerEmail := claimInstanceViaAPI(t, router)

	otherEmail := "other" + uuid.New().String() + "@example.com"
	signUpViaAPI(t, router, otherEmail)

	notAnAddress := "admin"
	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+owner.Token,
		users_dto.UpdateUserInfoRequestDTO{Email: &notAnAddress},
		http.StatusBadRequest,
	)

	response := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/me",
		"Bearer "+owner.Token,
		users_dto.UpdateUserInfoRequestDTO{Email: &otherEmail},
		http.StatusBadRequest,
	)
	assert.Contains(t, string(response.Body), "already taken")

	profile := getOwnProfile(t, router, owner.Token)
	assert.Equal(t, ownerEmail, profile.Email)
}

func Test_ResetPassword_WhenRequestedByBootstrapAdmin_PasswordIsReplaced(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	_, ownerEmail := claimInstanceViaAPI(t, router)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		users_dto.SendResetPasswordCodeRequestDTO{Email: ownerEmail},
		http.StatusOK,
	)

	require.Len(t, mockEmailSender.SentEmails, 1)
	assert.Equal(t, ownerEmail, mockEmailSender.SentEmails[0].To)

	code := extractCodeFromEmail(mockEmailSender.SentEmails[0].Body)
	require.Len(t, code, 6)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		users_dto.ResetPasswordRequestDTO{
			Email:       ownerEmail,
			Code:        code,
			NewPassword: "resetpassword123",
		},
		http.StatusOK,
	)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: ownerEmail, Password: "resetpassword123"},
		http.StatusOK,
	)
}

func Test_SignIn_WithUnknownAddressOnSingleUserInstance_DisclosesNothingAboutTheInstance(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()
	claimInstanceViaAPI(t, router)

	response := signInViaAPI(
		t,
		router,
		"unknown"+uuid.New().String()+"@example.com",
		http.StatusBadRequest,
	)

	assert.NotContains(t, string(response.Body), "admin")
	assert.NotContains(t, string(response.Body), "1")
}

func Test_SignIn_WhenAccountHasNoPassword_IsRefusedLikeAWrongPassword(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.RecreateInitialAdmin(t.Context())

	email := "oauth-only-" + uuid.New().String()[:8] + "@example.com"
	mockServer := newGitHubOAuthMockServer(email, int64(uuid.New().ID()))
	defer mockServer.Close()

	_, err := users_services.GetUserService().HandleGitHubOAuthWithMockEndpoint(
		t.Context(),
		"test-code",
		"http://localhost:3000/auth/callback",
		gitHubOAuthEndpoint(mockServer.URL),
		mockServer.URL+"/user",
	)
	require.NoError(t, err)

	response := signInViaAPI(t, router, email, http.StatusBadRequest)
	assert.Contains(t, string(response.Body), "password is incorrect")
}

func Test_SignIn_WhenAccountHasNotCompletedSignUp_IsRefused(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())

	inviter := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	invitedEmail := "invited" + uuid.New().String() + "@example.com"

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+inviter.Token,
		users_dto.InviteUserRequestDTO{Email: invitedEmail},
		http.StatusOK,
	)

	response := signInViaAPI(t, router, invitedEmail, http.StatusBadRequest)
	assert.Contains(t, string(response.Body), "not passed sign up yet")
}

func Test_SignIn_WhenAccountIsDeactivated_IsRefused(t *testing.T) {
	managementRouter := createManagementTestRouter()
	userRouter := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())

	rootAdmin := users_testing.RecreateInitAdminAndGetAccess(t.Context())
	memberEmail := "deactivated" + uuid.New().String() + "@example.com"
	member := signUpViaAPI(t, userRouter, memberEmail)

	test_utils.MakePostRequest(
		t,
		managementRouter,
		"/api/v1/users/"+member.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)

	response := signInViaAPI(t, userRouter, memberEmail, http.StatusBadRequest)
	assert.Contains(t, string(response.Body), "deactivated")
}

// Test_SignIn_WhenUpgradedInstanceStillUsesThePlaceholderAddress_Succeeds pins the
// half of the upgrade scenario a test can reach. SignInRequestDTO.Email binds
// `required` alone on purpose: adding `email` beside it, for symmetry with the
// registration DTO, would lock out every upgraded instance that has not yet moved
// off the placeholder. This test is what makes that edit fail.
func Test_SignIn_WhenUpgradedInstanceStillUsesThePlaceholderAddress_Succeeds(t *testing.T) {
	router := createUserTestRouter()
	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	repository := &users_repositories.UserRepository{}
	placeholderAdmin := &users_models.User{
		ID:                   uuid.New(),
		Email:                "admin",
		Name:                 "Admin",
		PasswordCreationTime: time.Now().UTC(),
		CreatedAt:            time.Now().UTC(),
		Role:                 users_enums.UserRoleAdmin,
		Status:               users_enums.UserStatusActive,
		IsRootAdmin:          true,
	}
	require.NoError(t, repository.CreateUser(placeholderAdmin))

	require.NoError(t, users_services.GetUserService().ChangeUserPasswordByEmail(
		t.Context(),
		"admin",
		bootstrapAdminPassword,
	))

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: "admin", Password: bootstrapAdminPassword},
		http.StatusOK,
	)
}

// claimInstanceViaAPI empties the instance and registers the account that takes
// it, returning a token for the bootstrap administrator and its address.
func claimInstanceViaAPI(t *testing.T, router *gin.Engine) (users_dto.SignInResponseDTO, string) {
	t.Helper()

	users_testing.ResetSettingsToDefaults(t.Context())
	users_testing.DeleteAllUsers()

	email := "owner" + uuid.New().String() + "@example.com"

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signup",
		"",
		users_dto.SignUpRequestDTO{
			Email:    email,
			Password: bootstrapAdminPassword,
			Name:     "Owner",
		},
		http.StatusOK,
		&response,
	)

	return response, email
}

func signInViaAPI(
	t *testing.T,
	router *gin.Engine,
	email string,
	expectedStatus int,
) *test_utils.TestResponse {
	t.Helper()

	return test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		users_dto.SignInRequestDTO{Email: email, Password: bootstrapAdminPassword},
		expectedStatus,
	)
}
