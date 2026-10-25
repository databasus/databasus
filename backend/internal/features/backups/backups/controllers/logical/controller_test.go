package backups_controllers_logical

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_download "databasus-backend/internal/features/backups/backups/download"
	"databasus-backend/internal/features/backups/backups/download/download_token"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
	local_storage "databasus-backend/internal/features/storages/models/local"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/tools"
)

func Test_GetBackups_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace viewer can get backups",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get backups",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get backups",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can get backups",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database, _, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
					workspaces_testing.AddMemberToWorkspace(
						workspace,
						member,
						*tt.workspaceRole,
						owner.Token,
						router,
					)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				testUserToken = nonMember.Token
			}

			testResp := test_utils.MakeGetRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/backups?database_id=%s", database.ID.String()),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				var response backups_dto_logical.GetBackupsResponse
				err := json.Unmarshal(testResp.Body, &response)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(response.Backups), 1)
				assert.GreaterOrEqual(t, response.Total, int64(1))
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}

			// Cleanup
			databases.RemoveTestDatabase(t.Context(), database)
			time.Sleep(50 * time.Millisecond)
			storages.RemoveTestStorage(t.Context(), storage.ID)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_GetBackups_WithStatusFilter_ReturnsFilteredBackups(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	now := time.Now().UTC()

	CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusCompleted,
		CreatedAt: now.Add(-3 * time.Hour),
	})
	CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusFailed,
		CreatedAt: now.Add(-2 * time.Hour),
	})
	CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusCanceled,
		CreatedAt: now.Add(-1 * time.Hour),
	})

	// Single status filter
	var singleResponse backups_dto_logical.GetBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups?database_id=%s&status=COMPLETED", database.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&singleResponse,
	)

	assert.Equal(t, int64(1), singleResponse.Total)
	assert.Len(t, singleResponse.Backups, 1)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, singleResponse.Backups[0].Status)

	// Multiple status filter
	var multiResponse backups_dto_logical.GetBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf(
			"/api/v1/backups?database_id=%s&status=COMPLETED&status=FAILED",
			database.ID.String(),
		),
		"Bearer "+owner.Token,
		http.StatusOK,
		&multiResponse,
	)

	assert.Equal(t, int64(2), multiResponse.Total)
	assert.Len(t, multiResponse.Backups, 2)

	for _, backup := range multiResponse.Backups {
		assert.True(
			t,
			backup.Status == backups_core_logical.BackupStatusCompleted ||
				backup.Status == backups_core_logical.BackupStatusFailed,
			"expected COMPLETED or FAILED, got %s", backup.Status,
		)
	}
}

func Test_GetBackups_WithBeforeDateFilter_ReturnsFilteredBackups(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	now := time.Now().UTC()
	cutoff := now.Add(-1 * time.Hour)

	olderBackup := CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusCompleted,
		CreatedAt: now.Add(-3 * time.Hour),
	})
	CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusCompleted,
		CreatedAt: now,
	})

	var response backups_dto_logical.GetBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf(
			"/api/v1/backups?database_id=%s&beforeDate=%s",
			database.ID.String(),
			cutoff.Format(time.RFC3339),
		),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, int64(1), response.Total)
	assert.Len(t, response.Backups, 1)
	assert.Equal(t, olderBackup.ID, response.Backups[0].ID)
}

func Test_GetBackups_WithCombinedFilters_ReturnsFilteredBackups(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	now := time.Now().UTC()
	cutoff := now.Add(-1 * time.Hour)

	// Old completed — should match
	oldCompleted := CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusCompleted,
		CreatedAt: now.Add(-3 * time.Hour),
	})
	// Old failed — should NOT match (wrong status)
	CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusFailed,
		CreatedAt: now.Add(-2 * time.Hour),
	})
	// New completed — should NOT match (too recent)
	CreateTestBackupWithOptions(database.ID, storage.ID, TestBackupOptions{
		Status:    backups_core_logical.BackupStatusCompleted,
		CreatedAt: now,
	})

	var response backups_dto_logical.GetBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf(
			"/api/v1/backups?database_id=%s&status=COMPLETED&beforeDate=%s",
			database.ID.String(),
			cutoff.Format(time.RFC3339),
		),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, int64(1), response.Total)
	assert.Len(t, response.Backups, 1)
	assert.Equal(t, oldCompleted.ID, response.Backups[0].ID)
}

func Test_CreateBackup_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can create backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can create backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can create backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot create backup",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can create backup",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
			enableBackupForDatabase(t.Context(), database.ID)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
					workspaces_testing.AddMemberToWorkspace(
						workspace,
						member,
						*tt.workspaceRole,
						owner.Token,
						router,
					)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				testUserToken = nonMember.Token
			}

			request := backups_dto_logical.MakeBackupRequest{DatabaseID: database.ID}
			testResp := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/backups",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "backup started successfully")
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}

			// Cleanup
			databases.RemoveTestDatabase(t.Context(), database)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_CreateBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	enableBackupForDatabase(t.Context(), database.ID)

	request := backups_dto_logical.MakeBackupRequest{DatabaseID: database.ID}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backups",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	time.Sleep(100 * time.Millisecond)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(t.Context(),
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup manually initiated") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for backup creation not found")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DeleteBackup_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can delete backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace member can delete backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace viewer cannot delete backup",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "non-member cannot delete backup",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can delete backup",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
					workspaces_testing.AddMemberToWorkspace(
						workspace,
						member,
						*tt.workspaceRole,
						owner.Token,
						router,
					)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				testUserToken = nonMember.Token
			}

			testResp := test_utils.MakeDeleteRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/backups/%s", backup.ID.String()),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
			)

			if !tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			} else {
				ownerActor := users.Actor{UserID: owner.UserID, Role: users.RoleMember}

				response, err := backups_services.GetBackupService().
					GetBackups(t.Context(), ownerActor, database.ID, 10, 0, nil)
				assert.NoError(t, err)
				assert.Equal(t, 0, len(response.Backups))
			}

			// Cleanup
			databases.RemoveTestDatabase(t.Context(), database)
			time.Sleep(50 * time.Millisecond)
			storages.RemoveTestStorage(t.Context(), storage.ID)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_DeleteBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s", backup.ID.String()),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	time.Sleep(100 * time.Millisecond)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(t.Context(),
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup deleted") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for backup deletion not found")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_GenerateDownloadToken_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace viewer can generate token",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can generate token",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot generate token",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can generate token",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil {
				if *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = owner.Token
				} else {
					member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
					workspaces_testing.AddMemberToWorkspace(
						workspace,
						member,
						*tt.workspaceRole,
						owner.Token,
						router,
					)
					testUserToken = member.Token
				}
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				testUserToken = nonMember.Token
			}

			testResp := test_utils.MakePostRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
				"Bearer "+testUserToken,
				nil,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				var response download_token.GenerateTokenResponse
				err := json.Unmarshal(testResp.Body, &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.NotEmpty(t, response.Filename)
				assert.Equal(t, backup.ID, response.BackupID)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}

			// Cleanup
			databases.RemoveTestDatabase(t.Context(), database)
			time.Sleep(50 * time.Millisecond)
			storages.RemoveTestStorage(t.Context(), storage.ID)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_DownloadBackup_WithValidToken_Success(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	// Generate download token
	var tokenResponse download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
		&tokenResponse,
	)

	// Download with token
	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), tokenResponse.Token),
		"",
		http.StatusOK,
	)

	// Verify response
	contentDisposition := testResp.Headers.Get("Content-Disposition")
	assert.Contains(t, contentDisposition, "attachment")
	assert.Contains(t, contentDisposition, tokenResponse.Filename)

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DownloadBackup_WithoutToken_Unauthorized(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	// Try to download without token
	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file", backup.ID.String()),
		"",
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(testResp.Body), "download token is required")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DownloadBackup_WithInvalidToken_Unauthorized(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	// Try to download with invalid token
	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), "invalid-token-xyz"),
		"",
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(testResp.Body), "invalid or expired download token")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DownloadBackup_WithExpiredToken_Unauthorized(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	// Get user for token generation
	userService := users_services.GetUserService()
	user, err := userService.GetUserFromToken(t.Context(), owner.Token)
	assert.NoError(t, err)

	// Create an expired token directly in the database
	expiredToken := createExpiredDownloadToken(
		t.Context(),
		expiredDownloadTokenSpec{BackupID: backup.ID, UserID: user.ID},
	)

	// Try to download with expired token
	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), expiredToken),
		"",
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(testResp.Body), "invalid or expired download token")

	// Verify audit log was NOT created for failed download
	time.Sleep(100 * time.Millisecond)
	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(t.Context(),
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup file downloaded") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.False(t, found, "Audit log should NOT be created for failed download with expired token")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DownloadBackup_TokenUsedOnce_CannotReuseToken(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	// Generate download token
	var tokenResponse download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
		&tokenResponse,
	)

	// Download with token (first time - should succeed)
	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), tokenResponse.Token),
		"",
		http.StatusOK,
	)

	// Try to download again with same token (should fail)
	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), tokenResponse.Token),
		"",
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(testResp.Body), "invalid or expired download token")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DownloadBackup_WithDifferentBackupToken_Unauthorized(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database1 := createTestDatabase("Database 1", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	configService := backups_config_logical.GetBackupConfigService()
	config1, err := configService.GetBackupConfigByDbId(database1.ID)
	assert.NoError(t, err)
	config1.IsBackupsEnabled = true
	config1.StorageID = &storage.ID
	config1.Storage = storage
	_, err = configService.SaveBackupConfig(t.Context(), config1)
	assert.NoError(t, err)

	backup1 := createTestBackup(t.Context(), database1, owner)

	database2 := createTestDatabase("Database 2", workspace.ID, owner.Token, router)
	config2, err := configService.GetBackupConfigByDbId(database2.ID)
	assert.NoError(t, err)
	config2.IsBackupsEnabled = true
	config2.StorageID = &storage.ID
	config2.Storage = storage
	_, err = configService.SaveBackupConfig(t.Context(), config2)
	assert.NoError(t, err)

	backup2 := createTestBackup(t.Context(), database2, owner)

	// Generate token for backup1
	var tokenResponse download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup1.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
		&tokenResponse,
	)

	// Try to use backup1's token to download backup2 (should fail)
	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup2.ID.String(), tokenResponse.Token),
		"",
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(testResp.Body), "invalid or expired download token")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database1)
	databases.RemoveTestDatabase(t.Context(), database2)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DownloadBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	// Generate download token
	var tokenResponse download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
		&tokenResponse,
	)

	// Download with token
	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), tokenResponse.Token),
		"",
		http.StatusOK,
	)

	time.Sleep(100 * time.Millisecond)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(t.Context(),
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup file downloaded") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for backup download not found")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DownloadBackup_ProperFilenameForPostgreSQL(t *testing.T) {
	tests := []struct {
		name           string
		databaseName   string
		expectedExt    string
		expectedInName string
	}{
		{
			name:           "PostgreSQL database",
			databaseName:   "my_postgres_db",
			expectedExt:    ".dump",
			expectedInName: "my_postgres_db_backup_",
		},
		{
			name:           "Database name with spaces",
			databaseName:   "my test db",
			expectedExt:    ".dump",
			expectedInName: "my_test_db_backup_",
		},
		{
			name:           "Database name with special characters",
			databaseName:   "my:db/test",
			expectedExt:    ".dump",
			expectedInName: "my-db-test_backup_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createTestDatabase(tt.databaseName, workspace.ID, owner.Token, router)
			storage := createTestStorage(workspace.ID)

			configService := backups_config_logical.GetBackupConfigService()
			config, err := configService.GetBackupConfigByDbId(database.ID)
			assert.NoError(t, err)

			config.IsBackupsEnabled = true
			config.StorageID = &storage.ID
			config.Storage = storage
			_, err = configService.SaveBackupConfig(t.Context(), config)
			assert.NoError(t, err)

			backup := createTestBackup(t.Context(), database, owner)

			// Generate download token
			var tokenResponse download_token.GenerateTokenResponse
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
				"Bearer "+owner.Token,
				nil,
				http.StatusOK,
				&tokenResponse,
			)

			// Download with token
			resp := test_utils.MakeGetRequest(
				t,
				router,
				fmt.Sprintf(
					"/api/v1/backups/%s/file?token=%s",
					backup.ID.String(),
					tokenResponse.Token,
				),
				"",
				http.StatusOK,
			)

			contentDisposition := resp.Headers.Get("Content-Disposition")
			assert.NotEmpty(t, contentDisposition, "Content-Disposition header should be present")

			// Verify the filename contains expected parts
			assert.Contains(
				t,
				contentDisposition,
				tt.expectedInName,
				"Filename should contain sanitized database name",
			)
			assert.Contains(
				t,
				contentDisposition,
				tt.expectedExt,
				"Filename should have correct extension",
			)
			assert.Contains(t, contentDisposition, "attachment", "Should be an attachment")

			// Verify timestamp format (YYYY-MM-DD_HH-mm-ss)
			assert.Regexp(
				t,
				`\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}`,
				contentDisposition,
				"Filename should contain timestamp",
			)

			// Cleanup
			databases.RemoveTestDatabase(t.Context(), database)
			time.Sleep(50 * time.Millisecond)
			storages.RemoveTestStorage(t.Context(), storage.ID)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_SanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "simple_name", expected: "simple_name"},
		{input: "name with spaces", expected: "name_with_spaces"},
		{input: "name/with\\slashes", expected: "name-with-slashes"},
		{input: "name:with*special?chars", expected: "name-with-special-chars"},
		{input: "name<with>pipes|", expected: "name-with-pipes-"},
		{input: `name"with"quotes`, expected: "name-with-quotes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := files_utils.SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_CancelBackup_InProgressBackup_SuccessfullyCancelled(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	configService := backups_config_logical.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	config.IsBackupsEnabled = true
	config.StorageID = &storage.ID
	config.Storage = storage
	_, err = configService.SaveBackupConfig(t.Context(), config)
	assert.NoError(t, err)

	backup := &backups_core_logical.LogicalBackup{
		ID:               uuid.New(),
		DatabaseID:       database.ID,
		StorageID:        storage.ID,
		Status:           backups_core_logical.BackupStatusInProgress,
		BackupSizeMb:     0,
		BackupDurationMs: 0,
		CreatedAt:        time.Now().UTC(),
	}

	repo := &backups_core_logical.BackupRepository{}
	err = repo.Save(backup)
	assert.NoError(t, err)

	// Register a cancellable context for the backup
	task_cancellation.GetRegistry().RegisterTask(backup.ID, func() {})

	resp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/cancel", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusNoContent,
	)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify audit log was created
	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetGlobalAuditLogs(t.Context(),
		users.Actor{UserID: admin.UserID, Role: users.RoleAdmin},
		&audit_logs.GetAuditLogsRequest{Limit: 100, Offset: 0},
	)
	assert.NoError(t, err)

	foundCancelLog := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Backup cancelled") &&
			strings.Contains(log.Message, database.Name) {
			foundCancelLog = true
			break
		}
	}
	assert.True(t, foundCancelLog, "Cancel audit log should be created")

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_ConcurrentDownloadPrevention(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	var token1Response download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
		&token1Response,
	)

	var token2Response download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
		&token2Response,
	)

	service := backups_services.GetBackupService()

	// Hold the per-user stream lock to simulate a download in progress, deterministically —
	// without racing a real download goroutine, which finishes too fast to observe under load.
	service.RefreshDownloadLock(t.Context(), owner.UserID)
	assert.True(t, service.IsDownloadInProgress(owner.UserID))

	resp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), token1Response.Token),
		"",
		http.StatusConflict,
	)

	var errorResponse map[string]string
	err := json.Unmarshal(resp.Body, &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "download already in progress")

	// After the lock is released, a download proceeds normally.
	service.ReleaseDownloadLock(t.Context(), owner.UserID)
	assert.False(t, service.IsDownloadInProgress(owner.UserID))

	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/file?token=%s", backup.ID.String(), token2Response.Token),
		"",
		http.StatusOK,
	)

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_GenerateDownloadToken_BlockedWhenDownloadInProgress(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, backup, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	service := backups_services.GetBackupService()

	// Mark a download in progress for this user deterministically. Taking the per-user stream
	// lock is exactly what a real download does; setting it directly avoids racing a real
	// download, which completes too fast to observe reliably under parallel test load.
	service.RefreshDownloadLock(t.Context(), owner.UserID)
	assert.True(t, service.IsDownloadInProgress(owner.UserID))

	resp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusConflict,
	)

	var errorResponse map[string]string
	err := json.Unmarshal(resp.Body, &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "Download already in progress")

	// Releasing the lock allows token generation again.
	service.ReleaseDownloadLock(t.Context(), owner.UserID)
	assert.False(t, service.IsDownloadInProgress(owner.UserID))

	var tokenResponse download_token.GenerateTokenResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s/download-token", backup.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
		&tokenResponse,
	)

	assert.NotEmpty(t, tokenResponse.Token)

	// Cleanup
	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_MakeBackup_VerifyBackupAndMetadataFilesExistInStorage(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database, _, storage := createTestDatabaseWithBackups(t.Context(), workspace, owner, router)

	scheduler := backuping_logical.CreateTestScheduler()
	schedulerCancel := backuping_logical.StartSchedulerForTest(t, scheduler)
	defer schedulerCancel()

	backupRepo := &backups_core_logical.BackupRepository{}
	initialBackups, err := backupRepo.FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	request := backups_dto_logical.MakeBackupRequest{DatabaseID: database.ID}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backups",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	backuping_logical.WaitForBackupCompletion(t, database.ID, len(initialBackups), 30*time.Second)

	backups, err := backupRepo.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Greater(t, len(backups), len(initialBackups))

	backup := backups[0]
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	storageService := storages.GetStorageService()
	backupStorage, err := storageService.GetStorageByID(t.Context(), backup.StorageID)
	assert.NoError(t, err)

	encryptor := encryption.GetFieldEncryptor()

	backupFile, err := backupStorage.GetFile(
		t.Context(),
		encryptor,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		backup.FileName,
	)
	require.NoError(t, err)
	backupFile.Close()

	metadataFile, err := backupStorage.GetFile(
		t.Context(),
		encryptor,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		backup.FileName+".metadata",
	)
	require.NoError(t, err)

	metadataContent, err := io.ReadAll(metadataFile)
	require.NoError(t, err)
	metadataFile.Close()

	var storageMetadata backups_core_logical.BackupMetadata
	err = json.Unmarshal(metadataContent, &storageMetadata)
	assert.NoError(t, err)

	assert.Equal(t, backup.ID, storageMetadata.BackupID)

	if backup.EncryptionSalt != nil && storageMetadata.EncryptionSalt != nil {
		assert.Equal(t, *backup.EncryptionSalt, *storageMetadata.EncryptionSalt)
	}

	if backup.EncryptionIV != nil && storageMetadata.EncryptionIV != nil {
		assert.Equal(t, *backup.EncryptionIV, *storageMetadata.EncryptionIV)
	}

	assert.Equal(t, backup.Encryption, storageMetadata.Encryption)

	err = backupRepo.DeleteByID(backup.ID)
	assert.NoError(t, err)

	databases.RemoveTestDatabase(t.Context(), database)
	time.Sleep(50 * time.Millisecond)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func createTestRouter() *gin.Engine {
	return CreateTestRouter()
}

func createTestDatabase(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *databases.Database {
	env := config.GetEnv()
	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse TEST_LOGICAL_POSTGRES_16_PORT: %v", err))
	}

	testDbName := "testdb"
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     config.GetEnv().TestLocalhost,
			Port:     port,
			Username: "testuser",
			Password: "testpassword",
			Database: &testDbName,
			CpuCount: 1,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		panic(
			fmt.Sprintf("Failed to create database. Status: %d, Body: %s", w.Code, w.Body.String()),
		)
	}

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}

	return &database
}

func createTestStorage(workspaceID uuid.UUID) *storages.Storage {
	storage := &storages.Storage{
		WorkspaceID:  workspaceID,
		Type:         storages.StorageTypeLocal,
		Name:         "Test Storage " + uuid.New().String(),
		LocalStorage: &local_storage.LocalStorage{},
	}

	repo := &storages.StorageRepository{}
	storage, err := repo.Save(storage)
	if err != nil {
		panic(err)
	}

	return storage
}

func enableBackupForDatabase(ctx context.Context, databaseID uuid.UUID) {
	configService := backups_config_logical.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		panic(err)
	}

	config.IsBackupsEnabled = true
	_, err = configService.SaveBackupConfig(ctx, config)
	if err != nil {
		panic(err)
	}
}

func createTestDatabaseWithBackups(
	ctx context.Context,
	workspace *workspaces_models.Workspace,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*databases.Database, *backups_core_logical.LogicalBackup, *storages.Storage) {
	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	configService := backups_config_logical.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(database.ID)
	if err != nil {
		panic(err)
	}

	config.IsBackupsEnabled = true
	config.StorageID = &storage.ID
	config.Storage = storage
	_, err = configService.SaveBackupConfig(ctx, config)
	if err != nil {
		panic(err)
	}

	backup := createTestBackup(ctx, database, owner)

	return database, backup, storage
}

func createTestBackup(
	ctx context.Context,
	database *databases.Database,
	owner *users_dto.SignInResponseDTO,
) *backups_core_logical.LogicalBackup {
	userService := users_services.GetUserService()
	user, err := userService.GetUserFromToken(ctx, owner.Token)
	if err != nil {
		panic(err)
	}

	loadedStorages, err := storages.GetStorageService().GetStorages(
		ctx,
		users.Actor{UserID: user.ID, Role: user.Role},
		*database.WorkspaceID,
	)
	if err != nil || len(loadedStorages) == 0 {
		panic("No storage found for workspace")
	}

	backup := &backups_core_logical.LogicalBackup{
		ID:               uuid.New(),
		DatabaseID:       database.ID,
		StorageID:        loadedStorages[0].ID,
		Status:           backups_core_logical.BackupStatusCompleted,
		BackupSizeMb:     10.5,
		BackupDurationMs: 1000,
		CreatedAt:        time.Now().UTC(),
	}

	repo := &backups_core_logical.BackupRepository{}
	if err := repo.Save(backup); err != nil {
		panic(err)
	}

	// Create a dummy backup file for testing download functionality
	dummyContent := []byte("dummy backup content for testing")
	reader := strings.NewReader(string(dummyContent))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := loadedStorages[0].SaveFile(
		context.Background(),
		encryption.GetFieldEncryptor(),
		logger,
		backup.ID.String(),
		reader,
	); err != nil {
		panic(fmt.Sprintf("Failed to create test backup file: %v", err))
	}

	return backup
}

type expiredDownloadTokenSpec struct {
	BackupID uuid.UUID
	UserID   uuid.UUID
}

func createExpiredDownloadToken(ctx context.Context, spec expiredDownloadTokenSpec) string {
	tokenService := backups_download.GetDownloadTokenService()
	token, err := tokenService.Generate(ctx, spec.BackupID, spec.UserID)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate download token: %v", err))
	}

	// Manually update the token to be expired
	repo := &download_token.Repository{}
	downloadToken, err := repo.FindByToken(ctx, token)
	if err != nil || downloadToken == nil {
		panic(fmt.Sprintf("Failed to find generated token: %v", err))
	}

	// Set expiration to 10 minutes ago
	downloadToken.ExpiresAt = time.Now().UTC().Add(-10 * time.Minute)
	if err := repo.Update(downloadToken); err != nil {
		panic(fmt.Sprintf("Failed to update token expiration: %v", err))
	}

	return token
}

func Test_DeleteBackup_RemovesBackupAndMetadataFilesFromDisk(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	configService := backups_config_logical.GetBackupConfigService()
	backupConfig, err := configService.GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	backupConfig.IsBackupsEnabled = true
	backupConfig.StorageID = &storage.ID
	backupConfig.Storage = storage
	_, err = configService.SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	scheduler := backuping_logical.CreateTestScheduler()
	schedulerCancel := backuping_logical.StartSchedulerForTest(t, scheduler)
	defer schedulerCancel()

	backupRepo := &backups_core_logical.BackupRepository{}
	initialBackups, err := backupRepo.FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	request := backups_dto_logical.MakeBackupRequest{DatabaseID: database.ID}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backups",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	backuping_logical.WaitForBackupCompletion(t, database.ID, len(initialBackups), 30*time.Second)

	backups, err := backupRepo.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Greater(t, len(backups), len(initialBackups))

	backup := backups[0]
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	dataFolder := config.GetEnv().DataFolder
	backupFilePath := filepath.Join(dataFolder, backup.FileName)
	metadataFilePath := filepath.Join(dataFolder, backup.FileName+".metadata")

	_, err = os.Stat(backupFilePath)
	assert.NoError(t, err, "backup file should exist on disk before deletion")

	_, err = os.Stat(metadataFilePath)
	assert.NoError(t, err, "metadata file should exist on disk before deletion")

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/backups/%s", backup.ID.String()),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	// Deleting a backup commits the obligation to remove its files; the worker
	// carries it out, so the test drives it before asserting the disk is clean.
	require.NoError(t, storages.DrainStorageFileDeletions(t.Context(),
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName},
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName + ".metadata"},
	))

	_, err = os.Stat(backupFilePath)
	assert.True(t, os.IsNotExist(err), "backup file should be removed from disk after deletion")

	_, err = os.Stat(metadataFilePath)
	assert.True(t, os.IsNotExist(err), "metadata file should be removed from disk after deletion")
}
