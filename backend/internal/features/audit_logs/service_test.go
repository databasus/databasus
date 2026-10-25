package audit_logs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
)

func Test_AuditLogs_WorkspaceSpecificLogs(t *testing.T) {
	service := GetAuditLogService()
	user1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace1ID, workspace2ID := uuid.New(), uuid.New()

	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace1 log first",
		UserID:      &user1.UserID,
		WorkspaceID: &workspace1ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace1 log second",
		UserID:      &user2.UserID,
		WorkspaceID: &workspace1ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace2 log first",
		UserID:      &user1.UserID,
		WorkspaceID: &workspace2ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace2 log second",
		UserID:      &user2.UserID,
		WorkspaceID: &workspace2ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test no workspace log",
		UserID:      &user1.UserID,
		WorkspaceID: nil,
	})

	request := &GetAuditLogsRequest{Limit: 10, Offset: 0}

	workspace1Response, err := service.GetWorkspaceAuditLogs(t.Context(), workspace1ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(workspace1Response.AuditLogs))

	messages := extractMessages(workspace1Response.AuditLogs)
	assert.Contains(t, messages, "Test workspace1 log first")
	assert.Contains(t, messages, "Test workspace1 log second")
	for _, log := range workspace1Response.AuditLogs {
		assert.Equal(t, &workspace1ID, log.WorkspaceID)
	}

	workspace2Response, err := service.GetWorkspaceAuditLogs(t.Context(), workspace2ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(workspace2Response.AuditLogs))

	messages2 := extractMessages(workspace2Response.AuditLogs)
	assert.Contains(t, messages2, "Test workspace2 log first")
	assert.Contains(t, messages2, "Test workspace2 log second")

	limitedResponse, err := service.GetWorkspaceAuditLogs(t.Context(), workspace1ID,
		&GetAuditLogsRequest{Limit: 1, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(limitedResponse.AuditLogs))
	assert.Equal(t, 1, limitedResponse.Limit)

	beforeTime := time.Now().UTC().Add(-1 * time.Minute)
	filteredResponse, err := service.GetWorkspaceAuditLogs(t.Context(), workspace1ID,
		&GetAuditLogsRequest{Limit: 10, BeforeDate: &beforeTime})
	assert.NoError(t, err)
	for _, log := range filteredResponse.AuditLogs {
		assert.True(t, log.CreatedAt.Before(beforeTime))
		assert.NotNil(t, log.UserEmail, "User email should be present for logs with user_id")
		assert.NotNil(
			t,
			log.WorkspaceName,
			"Workspace name should be present for logs with workspace_id",
		)
	}
}

func TestWriteAuditLog(t *testing.T) {
	t.Run("stores the author's email and name when the entry has a user ID", func(t *testing.T) {
		author := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Storage created: nightly-backups",
			UserID:      &author.UserID,
			WorkspaceID: &workspaceID,
		})

		storedLog := getSingleWorkspaceLog(t, workspaceID)
		assert.Equal(t, "Storage created: nightly-backups", storedLog.Message)
		assert.Equal(t, &author.UserID, storedLog.UserID)
		assert.Equal(t, &author.Email, storedLog.UserEmail)
		assert.NotNil(t, storedLog.UserName)
	})

	t.Run("keeps the author's email and name when the user changes both afterwards", func(t *testing.T) {
		author := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		authorBeforeChange, err := users_repositories.GetUserRepository().GetUserByID(t.Context(), author.UserID)
		require.NoError(t, err)
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Storage created: nightly-backups",
			UserID:      &author.UserID,
			WorkspaceID: &workspaceID,
		})
		renameUser(t, author.UserID)

		storedLog := getSingleWorkspaceLog(t, workspaceID)
		assert.Equal(t, &authorBeforeChange.Email, storedLog.UserEmail)
		assert.Equal(t, &authorBeforeChange.Name, storedLog.UserName)
	})

	t.Run("stores no author when the entry has no user ID", func(t *testing.T) {
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Backup failed: payments",
			UserID:      nil,
			WorkspaceID: &workspaceID,
		})

		storedLog := getSingleWorkspaceLog(t, workspaceID)
		assert.Nil(t, storedLog.UserID)
		assert.Nil(t, storedLog.UserEmail)
		assert.Nil(t, storedLog.UserName)
	})

	t.Run("stores the event without an author when the account no longer exists", func(t *testing.T) {
		removedUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		users_testing.DeleteTestUser(t.Context(), removedUser.UserID)
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Storage deleted: nightly-backups",
			UserID:      &removedUser.UserID,
			WorkspaceID: &workspaceID,
		})

		storedLog := getSingleWorkspaceLog(t, workspaceID)
		assert.Equal(t, "Storage deleted: nightly-backups", storedLog.Message)
		assert.Nil(t, storedLog.UserID)
	})
}

func getSingleWorkspaceLog(t *testing.T, workspaceID uuid.UUID) *AuditLogDTO {
	t.Helper()

	listedLogs, err := GetAuditLogService().GetWorkspaceAuditLogs(
		t.Context(), workspaceID, &GetAuditLogsRequest{Limit: 10, Offset: 0, BeforeDate: nil},
	)
	require.NoError(t, err)
	require.Len(t, listedLogs.AuditLogs, 1)

	return listedLogs.AuditLogs[0]
}

func renameUser(t *testing.T, userID uuid.UUID) {
	t.Helper()

	changedName := "Renamed " + uuid.NewString()
	changedEmail := "renamed-" + uuid.NewString() + "@example.com"
	require.NoError(t, users_repositories.GetUserRepository().UpdateUserInfo(userID, &changedName, &changedEmail))
}

func createAuditLog(t *testing.T, service *AuditLogService, entry audit_logs_models.AuditEntry) {
	t.Helper()

	service.WriteAuditLog(t.Context(), entry)
}

func extractMessages(logs []*AuditLogDTO) []string {
	messages := make([]string, len(logs))
	for i, log := range logs {
		messages[i] = log.Message
	}
	return messages
}
