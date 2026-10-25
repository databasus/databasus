package users_controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	users_dto "databasus-backend/internal/features/users/dto"
	users_models "databasus-backend/internal/features/users/models"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_SendResetPasswordCode_WithValidEmail_CodeSent(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.SendResetPasswordCodeRequestDTO{
		Email: user.Email,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		request,
		http.StatusOK,
	)

	assert.Equal(t, 1, len(mockEmailSender.SentEmails))
	assert.Equal(t, user.Email, mockEmailSender.SentEmails[0].To)
	assert.Contains(t, mockEmailSender.SentEmails[0].Subject, "Password Reset")
}

func Test_SendResetPasswordCode_WithNonExistentUser_ReturnsSuccess(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	request := users_dto.SendResetPasswordCodeRequestDTO{
		Email: "nonexistent" + uuid.New().String() + "@example.com",
	}

	// Should return success to prevent enumeration attacks
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		request,
		http.StatusOK,
	)

	// But no email should be sent
	assert.Equal(t, 0, len(mockEmailSender.SentEmails))
}

func Test_SendResetPasswordCode_WithInvitedUser_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	email := "invited" + uuid.New().String() + "@example.com"

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

	request := users_dto.SendResetPasswordCodeRequestDTO{
		Email: email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "only active users")
}

func Test_SendResetPasswordCode_WithRateLimitExceeded_ReturnsTooManyRequests(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.SendResetPasswordCodeRequestDTO{
		Email: user.Email,
	}

	// Make 3 requests (should succeed)
	for range 3 {
		test_utils.MakePostRequest(
			t,
			router,
			"/api/v1/users/send-reset-password-code",
			"",
			request,
			http.StatusOK,
		)
	}

	// 4th request should be rate limited
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		request,
		http.StatusTooManyRequests,
	)

	assert.Contains(t, string(resp.Body), "Rate limit exceeded")
}

func Test_SendResetPasswordCode_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/send-reset-password-code",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_ResetPassword_WithValidCode_PasswordReset(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	email := "resettest" + uuid.New().String() + "@example.com"
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"

	// Create user
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: oldPassword,
		Name:     "Test User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Request reset code
	sendCodeRequest := users_dto.SendResetPasswordCodeRequestDTO{
		Email: email,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		sendCodeRequest,
		http.StatusOK,
	)

	// Extract code from email
	assert.Equal(t, 1, len(mockEmailSender.SentEmails))
	emailBody := mockEmailSender.SentEmails[0].Body
	code := extractCodeFromEmail(emailBody)
	t.Logf("Extracted code: %s from email body (length: %d)", code, len(code))
	assert.NotEmpty(t, code, "Code should be extracted from email")
	assert.Len(t, code, 6, "Code should be 6 digits")

	// Reset password
	resetRequest := users_dto.ResetPasswordRequestDTO{
		Email:       email,
		Code:        code,
		NewPassword: newPassword,
	}
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		resetRequest,
		http.StatusOK,
	)
	if resp.StatusCode != http.StatusOK {
		t.Logf("Reset password failed with body: %s", string(resp.Body))
	}

	// Verify old password doesn't work
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

func Test_ResetPassword_WithExpiredCode_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// Create expired reset code directly in database
	code := "123456"
	hashedCode, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	expiredCode := &users_models.PasswordResetCode{
		ID:         uuid.New(),
		UserID:     user.UserID,
		HashedCode: string(hashedCode),
		ExpiresAt:  time.Now().UTC().Add(-1 * time.Hour), // Expired 1 hour ago
		IsUsed:     false,
		CreatedAt:  time.Now().UTC().Add(-2 * time.Hour),
	}
	storage.GetDb().Create(expiredCode)

	resetRequest := users_dto.ResetPasswordRequestDTO{
		Email:       user.Email,
		Code:        code,
		NewPassword: "newpassword123",
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		resetRequest,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "invalid or expired")
}

func Test_ResetPassword_WithUsedCode_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	email := "usedcode" + uuid.New().String() + "@example.com"

	// Create user
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "password123",
		Name:     "Test User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Request reset code
	sendCodeRequest := users_dto.SendResetPasswordCodeRequestDTO{
		Email: email,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		sendCodeRequest,
		http.StatusOK,
	)

	code := extractCodeFromEmail(mockEmailSender.SentEmails[0].Body)

	// Use code first time
	resetRequest := users_dto.ResetPasswordRequestDTO{
		Email:       email,
		Code:        code,
		NewPassword: "newpassword123",
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		resetRequest,
		http.StatusOK,
	)

	// Try to use same code again
	resetRequest2 := users_dto.ResetPasswordRequestDTO{
		Email:       email,
		Code:        code,
		NewPassword: "anotherpassword456",
	}
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		resetRequest2,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "invalid or expired")
}

func Test_ResetPassword_WithWrongCode_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	// Request reset code
	sendCodeRequest := users_dto.SendResetPasswordCodeRequestDTO{
		Email: user.Email,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		sendCodeRequest,
		http.StatusOK,
	)

	// Try to reset with wrong code
	resetRequest := users_dto.ResetPasswordRequestDTO{
		Email:       user.Email,
		Code:        "999999", // Wrong code
		NewPassword: "newpassword123",
	}
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		resetRequest,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "invalid")
}

func Test_ResetPassword_WithInvalidNewPassword_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	resetRequest := users_dto.ResetPasswordRequestDTO{
		Email:       user.Email,
		Code:        "123456",
		NewPassword: "short", // Too short
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		resetRequest,
		http.StatusBadRequest,
	)
}

func Test_ResetPassword_EmailSendFailure_ReturnsError(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	mockEmailSender.ShouldFail = true
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.SendResetPasswordCodeRequestDTO{
		Email: user.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "failed to send email")
}

func Test_ResetPasswordFlow_E2E_CompletesSuccessfully(t *testing.T) {
	router := createUserTestRouter()
	mockEmailSender := users_testing.NewMockEmailSender()
	users_services.GetUserService().SetEmailSender(mockEmailSender)

	email := "e2e" + uuid.New().String() + "@example.com"
	initialPassword := "initialpass123"
	newPassword := "brandnewpass456"

	// 1. Create user via signup
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: initialPassword,
		Name:     "E2E Test User",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// 2. Verify can sign in with initial password
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: initialPassword,
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
	assert.NotEmpty(t, signinResponse.Token)

	// 3. Request password reset code
	sendCodeRequest := users_dto.SendResetPasswordCodeRequestDTO{
		Email: email,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/send-reset-password-code",
		"",
		sendCodeRequest,
		http.StatusOK,
	)

	// 4. Verify email was sent
	assert.Equal(t, 1, len(mockEmailSender.SentEmails))
	code := extractCodeFromEmail(mockEmailSender.SentEmails[0].Body)
	assert.NotEmpty(t, code)

	// 5. Reset password using code
	resetRequest := users_dto.ResetPasswordRequestDTO{
		Email:       email,
		Code:        code,
		NewPassword: newPassword,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/reset-password",
		"",
		resetRequest,
		http.StatusOK,
	)

	// 6. Verify old password no longer works
	oldSignin := users_dto.SignInRequestDTO{
		Email:    email,
		Password: initialPassword,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signin",
		"",
		oldSignin,
		http.StatusBadRequest,
	)

	// 7. Verify new password works
	newSignin := users_dto.SignInRequestDTO{
		Email:    email,
		Password: newPassword,
	}
	var finalResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		newSignin,
		http.StatusOK,
		&finalResponse,
	)
	assert.NotEmpty(t, finalResponse.Token)
}

func Test_ResetPassword_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/reset-password",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

// Helper function to extract 6-digit code from email HTML body
func extractCodeFromEmail(emailBody string) string {
	// Look for pattern: <h1 ... >CODE</h1>
	// First find <h1
	h1Start := 0
	for i := range len(emailBody) - 3 {
		if emailBody[i:i+3] == "<h1" {
			h1Start = i
			break
		}
	}

	if h1Start == 0 {
		return ""
	}

	// Find the > after <h1
	contentStart := h1Start
	for i := h1Start; i < len(emailBody); i++ {
		if emailBody[i] == '>' {
			contentStart = i + 1
			break
		}
	}

	// Find </h1>
	contentEnd := contentStart
	for i := contentStart; i < len(emailBody)-5; i++ {
		if emailBody[i:i+5] == "</h1>" {
			contentEnd = i
			break
		}
	}

	if contentEnd <= contentStart {
		return ""
	}

	// Extract content and remove whitespace
	content := emailBody[contentStart:contentEnd]
	code := ""
	for i := range len(content) {
		if isDigit(content[i]) {
			code += string(content[i])
		}
	}

	if len(code) == 6 {
		return code
	}

	return ""
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
