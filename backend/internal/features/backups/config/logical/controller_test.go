package backups_config_logical

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/databases"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	local_storage "databasus-backend/internal/features/storages/models/local"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/period"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/tools"
)

func createTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetBackupConfigController(),
	)
	return router
}

func Test_SaveBackupConfig_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer cannot save backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can save backup config",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			defer func() {
				databases.RemoveTestDatabase(t.Context(), database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			}

			timeOfDay := "04:00"
			request := LogicalBackupConfig{
				DatabaseID:          database.ID,
				IsBackupsEnabled:    true,
				RetentionPolicyType: RetentionPolicyTypeTimePeriod,
				RetentionTimePeriod: period.PeriodWeek,
				BackupInterval: intervals.Interval{
					Type:      intervals.IntervalDaily,
					TimeOfDay: &timeOfDay,
				},
				SendNotificationsOn: []BackupNotificationType{
					NotificationBackupFailed,
				},
				IsRetryIfFailed:     true,
				MaxFailedTriesCount: 3,
			}

			var response LogicalBackupConfig
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/backup-configs/save",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.DatabaseID)
				assert.True(t, response.IsBackupsEnabled)
				assert.Equal(t, period.PeriodWeek, response.RetentionTimePeriod)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_SaveBackupConfig_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	timeOfDay := "04:00"
	request := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+nonMember.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_GetBackupConfigByDbID_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer can get backup config",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "global admin can get backup config",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get backup config",
			workspaceRole:      nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

			defer func() {
				databases.RemoveTestDatabase(t.Context(), database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				testUserToken = nonMember.Token
			}

			var response LogicalBackupConfig
			testResp := test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				"/api/v1/backup-configs/database/"+database.ID.String(),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.DatabaseID)
				assert.NotNil(t, response.BackupInterval)
			} else {
				assert.Contains(t, string(testResp.Body), "backup configuration not found")
			}
		})
	}
}

func Test_GetBackupConfigByDbID_ReturnsDefaultConfigForNewDatabase(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	var response LogicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.False(t, response.IsBackupsEnabled)
	assert.True(t, response.IsRetryIfFailed)
	assert.Equal(t, 3, response.MaxFailedTriesCount)
	assert.NotNil(t, response.BackupInterval)
}

func Test_IsStorageInUse_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		isStorageOwner     bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "storage owner can check storage usage",
			isStorageOwner:     true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-storage-owner cannot check storage usage",
			isStorageOwner:     false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			storageOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
				"Test Workspace",
				storageOwner,
				router,
			)
			storage := createTestStorage(workspace.ID)

			defer func() {
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			var testUserToken string
			if tt.isStorageOwner {
				testUserToken = storageOwner.Token
			} else {
				otherUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				testUserToken = otherUser.Token
			}

			if tt.expectSuccess {
				var response map[string]bool
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/backup-configs/storage/"+storage.ID.String()+"/is-using",
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
					&response,
				)

				isUsing, exists := response["isUsing"]
				assert.True(t, exists)
				assert.False(t, isUsing)
			} else {
				testResp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/backup-configs/storage/"+storage.ID.String()+"/is-using",
					"Bearer "+testUserToken,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(testResp.Body), "error")
			}
		})
	}
}

func Test_SaveBackupConfig_WithEncryptionNone_ConfigSaved(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	timeOfDay := "04:00"
	request := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	var response LogicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.Equal(t, backups_core_enums.BackupEncryptionNone, response.Encryption)
}

func Test_SaveBackupConfig_WithEncryptionEncrypted_ConfigSaved(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	timeOfDay := "04:00"
	request := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionEncrypted,
	}

	var response LogicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.Equal(t, backups_core_enums.BackupEncryptionEncrypted, response.Encryption)
}

func Test_TransferDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		sourceRole         *users_enums.WorkspaceRole
		targetRole         *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "owner in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "admin in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "member in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "viewer in both workspaces cannot transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "global admin can transfer",
			sourceRole:         nil,
			targetRole:         nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouterWithStorageForTransfer()

			sourceOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			targetOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

			sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
				"Source Workspace",
				sourceOwner,
				router,
			)
			targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
				"Target Workspace",
				targetOwner,
				router,
			)

			database := createTestDatabaseViaAPI(
				"Test Database",
				sourceWorkspace.ID,
				sourceOwner.Token,
				router,
			)

			targetStorage := createTestStorage(targetWorkspace.ID)

			defer func() {
				// Cleanup in correct order to avoid foreign key violations
				databases.RemoveTestDatabase(t.Context(), database)
				time.Sleep(50 * time.Millisecond) // Wait for cascade delete of backup_config
				storages.RemoveTestStorage(t.Context(), targetStorage.ID)
				workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
				workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
			}()

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.sourceRole != nil {
				testUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				if *tt.sourceRole == users_enums.WorkspaceRoleOwner {
					testUserToken = sourceOwner.Token
					workspaces_testing.AddMemberToWorkspace(
						targetWorkspace,
						sourceOwner,
						*tt.targetRole,
						targetOwner.Token,
						router,
					)
				} else {
					workspaces_testing.AddMemberToWorkspace(
						sourceWorkspace,
						testUser,
						*tt.sourceRole,
						sourceOwner.Token,
						router,
					)
					workspaces_testing.AddMemberToWorkspace(
						targetWorkspace,
						testUser,
						*tt.targetRole,
						targetOwner.Token,
						router,
					)
					testUserToken = testUser.Token
				}
			}

			request := TransferDatabaseRequest{
				TargetWorkspaceID: targetWorkspace.ID,
				TargetStorageID:   &targetStorage.ID,
			}

			testResp := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "database transferred successfully")

				var retrievedDatabase databases.Database
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/databases/"+database.ID.String(),
					"Bearer "+targetOwner.Token,
					http.StatusOK,
					&retrievedDatabase,
				)
				assert.Equal(t, targetWorkspace.ID, *retrievedDatabase.WorkspaceID)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_TransferDatabase_NonMemberInSourceWorkspace_CannotTransfer(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()

	sourceOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	targetOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"Source Workspace",
		sourceOwner,
		router,
	)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"Target Workspace",
		targetOwner,
		router,
	)

	database := createTestDatabaseViaAPI(
		"Test Database",
		sourceWorkspace.ID,
		sourceOwner.Token,
		router,
	)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	request := TransferDatabaseRequest{
		TargetWorkspaceID: targetWorkspace.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+targetOwner.Token,
		request,
		http.StatusForbidden,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_TransferDatabase_NonMemberInTargetWorkspace_CannotTransfer(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()

	sourceOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	targetOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"Source Workspace",
		sourceOwner,
		router,
	)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"Target Workspace",
		targetOwner,
		router,
	)

	database := createTestDatabaseViaAPI(
		"Test Database",
		sourceWorkspace.ID,
		sourceOwner.Token,
		router,
	)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	request := TransferDatabaseRequest{
		TargetWorkspaceID: targetWorkspace.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+sourceOwner.Token,
		request,
		http.StatusForbidden,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_TransferDatabase_ToNewStorage_DatabaseTransferd(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", sourceWorkspace.ID, owner.Token, router)
	sourceStorage := createTestStorage(sourceWorkspace.ID)
	targetStorage := createTestStorage(targetWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(200 * time.Millisecond) // Wait for cascading deletes
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: sourceStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	var savedConfig LogicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
		&savedConfig,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID: targetWorkspace.ID,
		TargetStorageID:   &targetStorage.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "database transferred successfully")

	var retrievedDatabase databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedDatabase,
	)
	assert.Equal(t, targetWorkspace.ID, *retrievedDatabase.WorkspaceID)

	var retrievedConfig LogicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedConfig,
	)
	assert.NotNil(t, retrievedConfig.StorageID)
	assert.Equal(t, targetStorage.ID, *retrievedConfig.StorageID)
}

func Test_TransferDatabase_WithExistingStorage_DatabaseAndStorageTransferd(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", sourceWorkspace.ID, owner.Token, router)
	storage := createTestStorage(sourceWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(200 * time.Millisecond) // Wait for cascading deletes
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: storage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	var savedConfig LogicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
		&savedConfig,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID:     targetWorkspace.ID,
		IsTransferWithStorage: true,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "database transferred successfully")

	var retrievedDatabase databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedDatabase,
	)
	assert.Equal(t, targetWorkspace.ID, *retrievedDatabase.WorkspaceID)

	var retrievedStorage storages.Storage
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/storages/"+storage.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedStorage,
	)
	assert.Equal(t, targetWorkspace.ID, retrievedStorage.WorkspaceID)
}

func Test_TransferDatabase_StorageHasOtherDBs_CannotTransfer(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database1 := createTestDatabaseViaAPI(
		"Test Database 1",
		sourceWorkspace.ID,
		owner.Token,
		router,
	)
	database2 := createTestDatabaseViaAPI(
		"Test Database 2",
		sourceWorkspace.ID,
		owner.Token,
		router,
	)
	storage := createTestStorage(sourceWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database1)
		databases.RemoveTestDatabase(t.Context(), database2)
		time.Sleep(200 * time.Millisecond) // Wait for cascading deletes
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	timeOfDay := "04:00"
	backupConfigRequest1 := LogicalBackupConfig{
		DatabaseID:          database1.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: storage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest1,
		http.StatusOK,
	)

	backupConfigRequest2 := LogicalBackupConfig{
		DatabaseID:          database2.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: storage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest2,
		http.StatusOK,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID:     targetWorkspace.ID,
		IsTransferWithStorage: true,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database1.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "storage has other attached databases")
}

func Test_TransferDatabase_WithNotifiers_NotifiersTransferred(t *testing.T) {
	router := createTestRouterWithNotifiersForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", sourceWorkspace.ID, owner.Token, router)
	sourceStorage := createTestStorage(sourceWorkspace.ID)
	targetStorage := createTestStorage(targetWorkspace.ID)
	notifier := notifiers.CreateTestNotifier(sourceWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(200 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	database.Notifiers = []notifiers.Notifier{*notifier}
	var updatedDatabase databases.Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database,
		http.StatusOK,
		&updatedDatabase,
	)

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: sourceStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID:       targetWorkspace.ID,
		TargetStorageID:         &targetStorage.ID,
		IsTransferWithNotifiers: true,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "database transferred successfully")

	var retrievedDatabase databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedDatabase,
	)
	assert.Equal(t, targetWorkspace.ID, *retrievedDatabase.WorkspaceID)
	assert.Len(t, retrievedDatabase.Notifiers, 1)

	var retrievedNotifier notifiers.Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers/"+notifier.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedNotifier,
	)
	assert.Equal(t, targetWorkspace.ID, retrievedNotifier.WorkspaceID)
}

func Test_TransferDatabase_NotifierHasOtherDBs_NotifierSkipped(t *testing.T) {
	router := createTestRouterWithNotifiersForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database1 := createTestDatabaseViaAPI(
		"Test Database 1",
		sourceWorkspace.ID,
		owner.Token,
		router,
	)
	database2 := createTestDatabaseViaAPI(
		"Test Database 2",
		sourceWorkspace.ID,
		owner.Token,
		router,
	)
	sourceStorage := createTestStorage(sourceWorkspace.ID)
	targetStorage := createTestStorage(targetWorkspace.ID)
	sharedNotifier := notifiers.CreateTestNotifier(sourceWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database1)
		databases.RemoveTestDatabase(t.Context(), database2)
		time.Sleep(200 * time.Millisecond)
		notifiers.RemoveTestNotifier(sharedNotifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	database1.Notifiers = []notifiers.Notifier{*sharedNotifier}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database1,
		http.StatusOK,
	)

	database2.Notifiers = []notifiers.Notifier{*sharedNotifier}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database2,
		http.StatusOK,
	)

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database1.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: sourceStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID:       targetWorkspace.ID,
		TargetStorageID:         &targetStorage.ID,
		IsTransferWithNotifiers: true,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database1.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "database transferred successfully")

	var retrievedDatabase databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database1.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedDatabase,
	)
	assert.Equal(t, targetWorkspace.ID, *retrievedDatabase.WorkspaceID)

	var retrievedNotifier notifiers.Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers/"+sharedNotifier.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedNotifier,
	)
	assert.Equal(t, sourceWorkspace.ID, retrievedNotifier.WorkspaceID)
}

func Test_TransferDatabase_WithMultipleNotifiers_OnlyExclusiveOnesTransferred(t *testing.T) {
	router := createTestRouterWithNotifiersForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database1 := createTestDatabaseViaAPI(
		"Test Database 1",
		sourceWorkspace.ID,
		owner.Token,
		router,
	)
	database2 := createTestDatabaseViaAPI(
		"Test Database 2",
		sourceWorkspace.ID,
		owner.Token,
		router,
	)
	sourceStorage := createTestStorage(sourceWorkspace.ID)
	targetStorage := createTestStorage(targetWorkspace.ID)

	exclusiveNotifier := notifiers.CreateTestNotifier(sourceWorkspace.ID)
	sharedNotifier := notifiers.CreateTestNotifier(sourceWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database1)
		databases.RemoveTestDatabase(t.Context(), database2)
		time.Sleep(200 * time.Millisecond)
		notifiers.RemoveTestNotifier(exclusiveNotifier)
		notifiers.RemoveTestNotifier(sharedNotifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	database1.Notifiers = []notifiers.Notifier{*exclusiveNotifier, *sharedNotifier}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database1,
		http.StatusOK,
	)

	database2.Notifiers = []notifiers.Notifier{*sharedNotifier}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database2,
		http.StatusOK,
	)

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database1.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: sourceStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID:       targetWorkspace.ID,
		TargetStorageID:         &targetStorage.ID,
		IsTransferWithNotifiers: true,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database1.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "database transferred successfully")

	var retrievedDatabase databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database1.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedDatabase,
	)
	assert.Equal(t, targetWorkspace.ID, *retrievedDatabase.WorkspaceID)
	assert.Len(t, retrievedDatabase.Notifiers, 2)

	var retrievedExclusiveNotifier notifiers.Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers/"+exclusiveNotifier.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedExclusiveNotifier,
	)
	assert.Equal(t, targetWorkspace.ID, retrievedExclusiveNotifier.WorkspaceID)

	var retrievedSharedNotifier notifiers.Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers/"+sharedNotifier.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedSharedNotifier,
	)
	assert.Equal(t, sourceWorkspace.ID, retrievedSharedNotifier.WorkspaceID)
}

func Test_TransferDatabase_WithTargetNotifiers_NotifiersAssigned(t *testing.T) {
	router := createTestRouterWithNotifiersForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", sourceWorkspace.ID, owner.Token, router)
	sourceStorage := createTestStorage(sourceWorkspace.ID)
	targetStorage := createTestStorage(targetWorkspace.ID)
	targetNotifier := notifiers.CreateTestNotifier(targetWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(200 * time.Millisecond)
		notifiers.RemoveTestNotifier(targetNotifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: sourceStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID: targetWorkspace.ID,
		TargetStorageID:   &targetStorage.ID,
		TargetNotifierIDs: []uuid.UUID{targetNotifier.ID},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "database transferred successfully")

	var retrievedDatabase databases.Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedDatabase,
	)
	assert.Equal(t, targetWorkspace.ID, *retrievedDatabase.WorkspaceID)
	assert.Len(t, retrievedDatabase.Notifiers, 1)
	assert.Equal(t, targetNotifier.ID, retrievedDatabase.Notifiers[0].ID)
}

func Test_TransferDatabase_TargetNotifierFromDifferentWorkspace_ReturnsBadRequest(t *testing.T) {
	router := createTestRouterWithNotifiersForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)
	otherWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Other Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", sourceWorkspace.ID, owner.Token, router)
	sourceStorage := createTestStorage(sourceWorkspace.ID)
	targetStorage := createTestStorage(targetWorkspace.ID)
	wrongNotifier := notifiers.CreateTestNotifier(otherWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(200 * time.Millisecond)
		notifiers.RemoveTestNotifier(wrongNotifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), otherWorkspace, router)
	}()

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: sourceStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID: targetWorkspace.ID,
		TargetStorageID:   &targetStorage.ID,
		TargetNotifierIDs: []uuid.UUID{wrongNotifier.ID},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "target notifier does not belong to target workspace")
}

func Test_TransferDatabase_TargetStorageFromDifferentWorkspace_ReturnsBadRequest(t *testing.T) {
	router := createTestRouterWithNotifiersForTransfer()

	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Source Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)
	otherWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Other Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", sourceWorkspace.ID, owner.Token, router)
	sourceStorage := createTestStorage(sourceWorkspace.ID)
	wrongStorage := createTestStorage(otherWorkspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(200 * time.Millisecond)
		workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), otherWorkspace, router)
	}()

	timeOfDay := "04:00"
	backupConfigRequest := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: sourceStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		backupConfigRequest,
		http.StatusOK,
	)

	request := TransferDatabaseRequest{
		TargetWorkspaceID: targetWorkspace.ID,
		TargetStorageID:   &wrongStorage.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/database/"+database.ID.String()+"/transfer",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "target storage does not belong to target workspace")
}

func Test_SaveBackupConfig_WithStorageFromAnotherWorkspace_ReturnsBadRequest(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()

	owner1 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspaceA := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace A", owner1, router)
	workspaceB := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace B", owner2, router)

	databaseA := createTestDatabaseViaAPI("Database A", workspaceA.ID, owner1.Token, router)

	storageB := createTestStorage(workspaceB.ID)

	timeOfDay := "04:00"
	backupConfigWithForeignStorage := LogicalBackupConfig{
		DatabaseID:          databaseA.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		StorageID: &storageB.ID,
		Storage:   storageB,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner1.Token,
		backupConfigWithForeignStorage,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "storage does not belong to the same workspace")

	// Cleanup: database first (cascades to backup_config), then storage, then workspaces
	databases.RemoveTestDatabase(t.Context(), databaseA)
	storages.RemoveTestStorage(t.Context(), storageB.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspaceA, router)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspaceB, router)
}

func createTestDatabaseViaAPI(
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
		WorkspaceID: &workspaceID,
		Name:        name,
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
	return storages.CreateTestStorage(workspaceID)
}

func createTestRouterWithStorageForTransfer() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetBackupConfigController(),
		storages.GetStorageController(),
	)

	storages.SetupDependencies()
	databases.SetupDependencies()
	SetupDependencies()

	return router
}

func createTestRouterWithNotifiersForTransfer() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetBackupConfigController(),
		storages.GetStorageController(),
		notifiers.GetNotifierController(),
	)

	storages.SetupDependencies()
	databases.SetupDependencies()
	notifiers.SetupDependencies()
	SetupDependencies()

	return router
}

func Test_SaveBackupConfig_WhenEmbeddedStorageHasChangedFields_KeepsStoredStorage(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	existingStorage := storages.CreateTestFlakyS3Storage(workspace.ID, "http://127.0.0.1:1")

	t.Cleanup(func() {
		databases.RemoveTestDatabase(t.Context(), database)
		storages.RemoveTestStorage(t.Context(), existingStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	})

	changedStorage := *existingStorage
	changedS3Storage := *existingStorage.S3Storage
	changedStorage.Name = "Renamed through the backup config"
	changedS3Storage.S3AccessKey = "replaced-access-key"
	changedS3Storage.S3SecretKey = "replaced-secret-key"
	changedStorage.S3Storage = &changedS3Storage

	var savedConfig LogicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		newLogicalConfigWithStorage(database.ID, &changedStorage),
		http.StatusOK,
		&savedConfig,
	)

	require.NotNil(t, savedConfig.StorageID)
	assert.Equal(t, existingStorage.ID, *savedConfig.StorageID)

	storedStorage, err := storages.GetStorageService().GetStorageByID(t.Context(), existingStorage.ID)
	require.NoError(t, err)
	assert.Equal(t, existingStorage.Name, storedStorage.Name)
	assert.Equal(t, existingStorage.S3Storage.S3AccessKey, storedStorage.S3Storage.S3AccessKey)
	assert.Equal(t, existingStorage.S3Storage.S3SecretKey, storedStorage.S3Storage.S3SecretKey)
}

func Test_SaveBackupConfig_WhenEmbeddedStorageHasNoID_ReturnsBadRequestAndCreatesNoStorage(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	t.Cleanup(func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	})

	storageWithoutID := &storages.Storage{
		WorkspaceID:  workspace.ID,
		Type:         storages.StorageTypeLocal,
		Name:         "Storage created through the backup config",
		LocalStorage: &local_storage.LocalStorage{},
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		newLogicalConfigWithStorage(database.ID, storageWithoutID),
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), ErrStorageWithoutID.Error())
	assert.Zero(t, countStoragesInWorkspace(t, workspace.ID))
}

func Test_SaveBackupConfig_WhenEmbeddedStorageIsUnknown_ReturnsBadRequestAndCreatesNoStorage(t *testing.T) {
	router := createTestRouterWithStorageForTransfer()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	t.Cleanup(func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	})

	unknownStorage := &storages.Storage{
		ID:           uuid.New(),
		WorkspaceID:  workspace.ID,
		Type:         storages.StorageTypeLocal,
		Name:         "Unknown storage",
		LocalStorage: &local_storage.LocalStorage{},
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		newLogicalConfigWithStorage(database.ID, unknownStorage),
		http.StatusBadRequest,
	)

	var storagesWithUnknownID int64
	require.NoError(t, storage.GetDb().Model(&storages.Storage{}).
		Where("id = ?", unknownStorage.ID).
		Count(&storagesWithUnknownID).Error)
	assert.Zero(t, storagesWithUnknownID)
}

func newLogicalConfigWithStorage(databaseID uuid.UUID, embeddedStorage *storages.Storage) LogicalBackupConfig {
	timeOfDay := "04:00"

	return LogicalBackupConfig{
		DatabaseID:          databaseID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: embeddedStorage,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}
}

func countStoragesInWorkspace(t *testing.T, workspaceID uuid.UUID) int64 {
	t.Helper()

	var storagesInWorkspace int64
	require.NoError(t, storage.GetDb().Model(&storages.Storage{}).
		Where("workspace_id = ?", workspaceID).
		Count(&storagesInWorkspace).Error)

	return storagesInWorkspace
}
