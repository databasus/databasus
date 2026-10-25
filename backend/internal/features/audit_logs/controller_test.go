package audit_logs

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_GetGlobalAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly(t *testing.T) {
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	memberUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	router := createRouter()
	service := GetAuditLogService()
	workspaceID := uuid.New()
	testID := uuid.New().String()

	// Create test logs with unique identifiers
	userLogMessage := fmt.Sprintf("Test log with user %s", testID)
	workspaceLogMessage := fmt.Sprintf("Test log with workspace %s", testID)
	standaloneLogMessage := fmt.Sprintf("Test log standalone %s", testID)

	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     userLogMessage,
		UserID:      &adminUser.UserID,
		WorkspaceID: nil,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     workspaceLogMessage,
		UserID:      nil,
		WorkspaceID: &workspaceID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     standaloneLogMessage,
		UserID:      nil,
		WorkspaceID: nil,
	})

	var response GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/audit-logs/global?limit=100", "Bearer "+adminUser.Token, http.StatusOK, &response)

	// Verify our specific test logs are present
	messages := extractMessages(response.AuditLogs)
	assert.Contains(t, messages, userLogMessage)
	assert.Contains(t, messages, workspaceLogMessage)
	assert.Contains(t, messages, standaloneLogMessage)

	// Test MEMBER cannot access global logs
	resp := test_utils.MakeGetRequest(t, router, "/api/v1/audit-logs/global",
		"Bearer "+memberUser.Token, http.StatusForbidden)
	assert.Contains(t, string(resp.Body), "only administrators can view global audit logs")
}

func Test_GetUserAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly(t *testing.T) {
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	user1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	router := createRouter()
	service := GetAuditLogService()
	workspaceID := uuid.New()
	testID := uuid.New().String()

	// Create test logs for different users with unique identifiers
	user1FirstMessage := fmt.Sprintf("Test log user1 first %s", testID)
	user1SecondMessage := fmt.Sprintf("Test log user1 second %s", testID)
	user2FirstMessage := fmt.Sprintf("Test log user2 first %s", testID)
	user2SecondMessage := fmt.Sprintf("Test log user2 second %s", testID)
	workspaceLogMessage := fmt.Sprintf("Test workspace log %s", testID)

	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     user1FirstMessage,
		UserID:      &user1.UserID,
		WorkspaceID: nil,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     user1SecondMessage,
		UserID:      &user1.UserID,
		WorkspaceID: &workspaceID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     user2FirstMessage,
		UserID:      &user2.UserID,
		WorkspaceID: nil,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     user2SecondMessage,
		UserID:      &user2.UserID,
		WorkspaceID: &workspaceID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     workspaceLogMessage,
		UserID:      nil,
		WorkspaceID: &workspaceID,
	})

	// Test ADMIN can view any user's logs
	var user1Response GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		fmt.Sprintf("/api/v1/audit-logs/users/%s?limit=100", user1.UserID.String()),
		"Bearer "+adminUser.Token, http.StatusOK, &user1Response)

	// Verify user1's specific logs are present
	messages := extractMessages(user1Response.AuditLogs)
	assert.Contains(t, messages, user1FirstMessage)
	assert.Contains(t, messages, user1SecondMessage)

	// Count only our test logs for user1
	testLogsCount := 0
	for _, message := range messages {
		if message == user1FirstMessage || message == user1SecondMessage {
			testLogsCount++
		}
	}
	assert.Equal(t, 2, testLogsCount)

	// Test user can view own logs
	var ownLogsResponse GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		fmt.Sprintf("/api/v1/audit-logs/users/%s?limit=100", user2.UserID.String()),
		"Bearer "+user2.Token, http.StatusOK, &ownLogsResponse)

	// Verify user2's specific logs are present
	ownMessages := extractMessages(ownLogsResponse.AuditLogs)
	assert.Contains(t, ownMessages, user2FirstMessage)
	assert.Contains(t, ownMessages, user2SecondMessage)

	// Test user cannot view other user's logs
	resp := test_utils.MakeGetRequest(t, router,
		fmt.Sprintf("/api/v1/audit-logs/users/%s", user1.UserID.String()),
		"Bearer "+user2.Token, http.StatusForbidden)

	assert.Contains(t, string(resp.Body), "insufficient permissions")
}

func Test_GetGlobalAuditLogs_WithBeforeDateFilter_ReturnsFilteredLogs(t *testing.T) {
	adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	router := createRouter()
	baseTime := time.Now().UTC()

	// Set filter time to 30 minutes ago
	beforeTime := baseTime.Add(-30 * time.Minute)

	var filteredResponse GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf(
			"/api/v1/audit-logs/global?beforeDate=%s&limit=1000",
			beforeTime.Format(time.RFC3339),
		),
		"Bearer "+adminUser.Token,
		http.StatusOK,
		&filteredResponse,
	)

	// Verify ALL returned logs are older than the filter time
	for _, log := range filteredResponse.AuditLogs {
		assert.True(t, log.CreatedAt.Before(beforeTime),
			fmt.Sprintf("Log created at %s should be before filter time %s",
				log.CreatedAt.Format(time.RFC3339), beforeTime.Format(time.RFC3339)))
	}
}

func TestGetAuditLogs(t *testing.T) {
	t.Run("shows the author's email and name from the event when the user changed both afterwards",
		func(t *testing.T) {
			adminUser := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
			author := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			authorBeforeChange, err := users_repositories.GetUserRepository().GetUserByID(t.Context(), author.UserID)
			require.NoError(t, err)
			router := createRouter()
			message := "Storage created: " + uuid.NewString()

			createAuditLog(t, GetAuditLogService(), audit_logs_models.AuditEntry{
				Message:     message,
				UserID:      &author.UserID,
				WorkspaceID: nil,
			})
			renameUser(t, author.UserID)

			for _, path := range []string{
				"/api/v1/audit-logs/global?limit=1000",
				fmt.Sprintf("/api/v1/audit-logs/users/%s?limit=100", author.UserID),
			} {
				var listedLogs GetAuditLogsResponse
				test_utils.MakeGetRequestAndUnmarshal(t, router, path, "Bearer "+adminUser.Token, http.StatusOK,
					&listedLogs)

				listedLog := findLogWithMessage(t, listedLogs.AuditLogs, message)
				assert.Equal(t, &authorBeforeChange.Email, listedLog.UserEmail, path)
				assert.Equal(t, &authorBeforeChange.Name, listedLog.UserName, path)
			}
		},
	)
}

func findLogWithMessage(t *testing.T, listedLogs []*AuditLogDTO, message string) *AuditLogDTO {
	t.Helper()

	for _, listedLog := range listedLogs {
		if listedLog.Message == message {
			return listedLog
		}
	}

	require.Failf(t, "audit log not listed", "no log with message %q", message)

	return nil
}

func createRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupDependencies()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetAuditLogController().RegisterRoutes(protected.(*gin.RouterGroup))

	return router
}
