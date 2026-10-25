package users_controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_dto "databasus-backend/internal/features/users/dto"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_GetUserSettings_WhenUserIsAdmin_ReturnsSettings(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var response users_dto.SettingsResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	// Default settings should be true for all
	assert.True(t, response.IsAllowExternalRegistrations)
	assert.True(t, response.IsAllowMemberInvitations)
	assert.True(t, response.IsMemberAllowedToCreateWorkspaces)

	// The second factor is the exception: an instance asks for one factor until
	// an administrator says otherwise.
	assert.False(t, response.IsTwoFactorAuthRequired)
}

func Test_GetUserSettings_WhenUserIsMember_ReturnsSettings(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	_ = test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		http.StatusOK,
	)
}

func Test_GetUserSettings_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	test_utils.MakeGetRequest(t, router, "/api/v1/users/settings", "", http.StatusUnauthorized)
}

func Test_UpdateUserSettings_WhenUserIsAdmin_SettingsUpdated(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Update some settings
	request := users_dto.UpdateSettingsRequestDTO{
		IsAllowExternalRegistrations:      false,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: false,
	}

	var response users_dto.SettingsResponseDTO
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		request,
		http.StatusOK,
		&response,
	)

	// Check that settings were updated
	assert.False(t, response.IsAllowExternalRegistrations)
	assert.True(t, response.IsAllowMemberInvitations)
	assert.False(t, response.IsMemberAllowedToCreateWorkspaces)
}

func Test_UpdateUserSettings_WithPartialData_SettingsUpdated(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Update only one setting
	request := users_dto.UpdateSettingsRequestDTO{
		IsAllowExternalRegistrations: false,
		// Other fields will use default values
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: true,
	}

	var response users_dto.SettingsResponseDTO
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		request,
		http.StatusOK,
		&response,
	)

	// Check that only the specified setting was updated
	assert.False(t, response.IsAllowExternalRegistrations)
	// These should remain true (default values)
	assert.True(t, response.IsAllowMemberInvitations)
	assert.True(t, response.IsMemberAllowedToCreateWorkspaces)
}

func Test_UpdateUserSettings_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := users_dto.UpdateSettingsRequestDTO{
		IsAllowExternalRegistrations: false,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "Insufficient permissions")
}

func Test_UpdateUserSettings_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/settings",
		Body:           "invalid json",
		AuthToken:      "Bearer " + testUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_UpdateUserSettings_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router := createSettingsTestRouter()

	request := users_dto.UpdateSettingsRequestDTO{
		IsAllowExternalRegistrations: false,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"",
		request,
		http.StatusUnauthorized,
	)
}

func Test_GetUserSettings_WhenMailServerIsConfigured_ReportsItConfigured(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, false)

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var response users_dto.SettingsResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.IsEmailConfigured)
}

func Test_GetUserSettings_WhenMailServerIsMissing_ReportsItUnconfigured(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, true)

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var response users_dto.SettingsResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		http.StatusOK,
		&response,
	)

	assert.False(t, response.IsEmailConfigured)
}

func Test_UpdateUserSettings_WhenRequestCarriesTheDerivedField_StoredRowIsUntouched(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, true)

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	response := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "PUT",
		URL:    "/api/v1/users/settings",
		Body: map[string]any{
			"isAllowExternalRegistrations":      true,
			"isAllowMemberInvitations":          true,
			"isMemberAllowedToCreateWorkspaces": true,
			"isTwoFactorAuthRequired":           false,
			"isEmailConfigured":                 true,
		},
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusOK,
	})

	var updateResponse users_dto.SettingsResponseDTO
	require.NoError(t, json.Unmarshal(response.Body, &updateResponse))
	assert.False(t, updateResponse.IsEmailConfigured)

	storedSettings, err := users_services.GetSettingsService().GetSettings(t.Context())
	require.NoError(t, err)
	assert.False(t, storedSettings.IsTwoFactorAuthRequired)
}

func Test_UpdateUserSettings_TurningTwoFactorOn_ReadsBackTheNewValue(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, false)
	users_testing.DeleteAllUsers()

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var beforeUpdate users_dto.SettingsResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		http.StatusOK,
		&beforeUpdate,
	)
	assert.False(t, beforeUpdate.IsTwoFactorAuthRequired)

	var afterUpdate users_dto.SettingsResponseDTO
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		enableTwoFactorRequest(),
		http.StatusOK,
		&afterUpdate,
	)
	assert.True(t, afterUpdate.IsTwoFactorAuthRequired)

	var readBack users_dto.SettingsResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		http.StatusOK,
		&readBack,
	)
	assert.True(t, readBack.IsTwoFactorAuthRequired)
}

func Test_UpdateUserSettings_WhenASwitchChanges_AuditLogCarriesItsOldAndNewValue(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, false)
	users_testing.DeleteAllUsers()

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	recorder := users_testing.GetAuditLogRecorder()

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		users_dto.UpdateSettingsRequestDTO{
			IsAllowExternalRegistrations:      false,
			IsAllowMemberInvitations:          false,
			IsMemberAllowedToCreateWorkspaces: false,
			IsTwoFactorAuthRequired:           true,
		},
		http.StatusOK,
	)

	expectedEntries := []string{
		"isAllowExternalRegistrations: true -> false",
		"isAllowMemberInvitations: true -> false",
		"isMemberAllowedToCreateWorkspaces: true -> false",
		"isTwoFactorAuthRequired: false -> true",
	}

	for _, expected := range expectedEntries {
		assert.Equal(t, 1, countAuditEntries(recorder, expected), expected)
	}
}

func Test_UpdateUserSettings_WhenNothingChanges_WritesNoAuditEntry(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, false)

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	recorder := users_testing.GetAuditLogRecorder()

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		users_dto.UpdateSettingsRequestDTO{
			IsAllowExternalRegistrations:      true,
			IsAllowMemberInvitations:          true,
			IsMemberAllowedToCreateWorkspaces: true,
			IsTwoFactorAuthRequired:           false,
		},
		http.StatusOK,
	)

	assert.Equal(t, 0, countAuditEntries(recorder, "isTwoFactorAuthRequired"))
	assert.Equal(t, 0, countAuditEntries(recorder, "isAllowExternalRegistrations"))
}

func Test_UpdateUserSettings_WhenMailServerIsMissing_RefusesToRequireTwoFactor(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, true)
	users_testing.DeleteAllUsers()

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	response := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		enableTwoFactorRequest(),
		http.StatusBadRequest,
	)
	assert.Contains(t, string(response.Body), "no mail server configured")

	storedSettings, err := users_services.GetSettingsService().GetSettings(t.Context())
	require.NoError(t, err)
	assert.False(t, storedSettings.IsTwoFactorAuthRequired)
}

func Test_UpdateUserSettings_WhenAnAdminHasNoUsableAddress_RefusesAndNamesTheAccount(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, false)
	users_testing.DeleteAllUsers()

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	unreachableAdmin := users_testing.CreateTestUserWithEmail(
		t.Context(),
		users.RoleAdmin,
		"admin",
	)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), unreachableAdmin.UserID) })

	response := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		enableTwoFactorRequest(),
		http.StatusBadRequest,
	)
	assert.Contains(t, string(response.Body), "admin")
	assert.Contains(t, string(response.Body), "no valid email address")

	storedSettings, err := users_services.GetSettingsService().GetSettings(t.Context())
	require.NoError(t, err)
	assert.False(t, storedSettings.IsTwoFactorAuthRequired)
}

func Test_UpdateUserSettings_WhenAnInactiveAdminHasNoUsableAddress_StillAccepts(t *testing.T) {
	users_testing.ResetSettingsToDefaults(t.Context())
	router, _ := createSettingsTestRouterWithMailSender(t, false)
	users_testing.DeleteAllUsers()

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	retiredAdmin := users_testing.CreateTestUserWithEmail(
		t.Context(),
		users.RoleAdmin,
		"retired-admin",
	)
	users_testing.DeactivateTestUser(t.Context(), retiredAdmin.UserID)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), retiredAdmin.UserID) })

	var response users_dto.SettingsResponseDTO
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+admin.Token,
		enableTwoFactorRequest(),
		http.StatusOK,
		&response,
	)

	assert.True(t, response.IsTwoFactorAuthRequired)
}

func enableTwoFactorRequest() users_dto.UpdateSettingsRequestDTO {
	return users_dto.UpdateSettingsRequestDTO{
		IsAllowExternalRegistrations:      true,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: true,
		IsTwoFactorAuthRequired:           true,
	}
}

func countAuditEntries(recorder *users_testing.RecordingAuditLogWriter, fragment string) int {
	count := 0

	for _, entry := range recorder.GetEntries() {
		if strings.Contains(entry.Message, fragment) {
			count++
		}
	}

	return count
}

func Test_SendTestEmail_WhenAdminHasAddress_SendsToAdminAndAuditsDelivery(t *testing.T) {
	router, mockEmailSender := createSettingsTestRouterWithMailSender(t, false)
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var response users_dto.SendTestEmailResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/users/settings/test-email", "Bearer "+admin.Token, nil, http.StatusOK, &response,
	)

	assert.Equal(t, admin.Email, response.RecipientEmail)
	require.Len(t, mockEmailSender.SentEmails, 1)
	assert.Equal(t, admin.Email, mockEmailSender.SentEmails[0].To)
	assert.True(t, users_testing.GetAuditLogRecorder().HasEntryContaining("Test email sent to", admin.Email))
}

func Test_SendTestEmail_WhenUserIsMember_ReturnsForbiddenAndSendsNothing(t *testing.T) {
	router, mockEmailSender := createSettingsTestRouterWithMailSender(t, false)
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	test_utils.MakePostRequest(
		t, router, "/api/v1/users/settings/test-email", "Bearer "+member.Token, nil, http.StatusForbidden,
	)

	assert.Empty(t, mockEmailSender.SentEmails)
}

func Test_SendTestEmail_WhenMailServerIsMissing_RefusesWithCodeAndSendsNothing(t *testing.T) {
	router, mockEmailSender := createSettingsTestRouterWithMailSender(t, true)
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/users/settings/test-email", "Bearer "+admin.Token, nil, http.StatusBadRequest,
	)

	assert.Equal(t, "email_not_configured", readErrorCode(t, response))
	assert.Empty(t, mockEmailSender.SentEmails)
}

func Test_SendTestEmail_WhenAdminHasBootstrapLogin_RefusesWithCodeAndSendsNothing(t *testing.T) {
	router, mockEmailSender := createSettingsTestRouterWithMailSender(t, false)
	bootstrapAdmin := users_testing.CreateTestUserWithEmail(t.Context(), users.RoleAdmin, "admin")
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), bootstrapAdmin.UserID) })

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/users/settings/test-email", "Bearer "+bootstrapAdmin.Token, nil, http.StatusBadRequest,
	)

	assert.Equal(t, "admin_email_missing", readErrorCode(t, response))
	assert.Empty(t, mockEmailSender.SentEmails)
}

func Test_SendTestEmail_SixthRequestWithinWindow_ReturnsTooManyRequests(t *testing.T) {
	router, mockEmailSender := createSettingsTestRouterWithMailSender(t, false)
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	for range 5 {
		test_utils.MakePostRequest(
			t, router, "/api/v1/users/settings/test-email", "Bearer "+admin.Token, nil, http.StatusOK,
		)
	}

	test_utils.MakePostRequest(
		t, router, "/api/v1/users/settings/test-email", "Bearer "+admin.Token, nil, http.StatusTooManyRequests,
	)

	assert.Len(t, mockEmailSender.SentEmails, 5)
}

func Test_SendTestEmail_WhenDeliveryFails_ReturnsSenderErrorAndAuditsFailure(t *testing.T) {
	router, mockEmailSender := createSettingsTestRouterWithMailSender(t, false)
	mockEmailSender.ShouldFail = true
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/users/settings/test-email", "Bearer "+admin.Token, nil, http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "mock email send failure")
	assert.Empty(t, readErrorCode(t, response))
	assert.True(t, users_testing.GetAuditLogRecorder().HasEntryContaining("Test email to", admin.Email, "failed"))
}

func readErrorCode(t *testing.T, response *test_utils.TestResponse) string {
	t.Helper()

	var errorResponse struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(response.Body, &errorResponse))

	return errorResponse.Code
}
