package databases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/audit_logs"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_models "databasus-backend/internal/features/users/models"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/encryption"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
	"databasus-backend/internal/util/walmath"
)

func Test_CreateDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can create database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace member can create database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace viewer cannot create database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can create database",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

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

			request := Database{
				Name:              "Test Database",
				WorkspaceID:       &workspace.ID,
				Type:              DatabaseTypePostgresLogical,
				PostgresqlLogical: getTestPostgresConfig(),
			}

			var response Database
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/create",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				defer RemoveTestDatabase(t.Context(), &response)
				assert.Equal(t, "Test Database", response.Name)
				assert.NotEqual(t, uuid.Nil, response.ID)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_CreateDatabase_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	request := Database{
		Name:              "Test Database",
		WorkspaceID:       &workspace.ID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+nonMember.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_CreateDatabase_WithoutConnectionFields_ValidationFails(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	request := Database{
		Name:        "Test Database",
		WorkspaceID: &workspace.ID,
		Type:        DatabaseTypePostgresLogical,
		PostgresqlLogical: &postgresql_logical.PostgresqlLogicalDatabase{
			CpuCount: 1,
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "host is required")
}

func Test_UpdateDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can update database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can update database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer cannot update database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can update database",
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
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(t.Context(), database)

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

			database.Name = "Updated Database"

			var response Database
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/update",
				"Bearer "+testUserToken,
				database,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, "Updated Database", response.Name)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_UpdateDatabase_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(t.Context(), database)

	nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	database.Name = "Hacked Name"

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+nonMember.Token,
		database,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_UpdateDatabase_WhenDatabaseTypeChanged_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(t.Context(), database)

	database.Type = DatabaseTypeMysql

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "database type cannot be changed")
}

func Test_DeleteDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can delete database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace member can delete database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "workspace viewer cannot delete database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "global admin can delete database",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

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

			testResp := test_utils.MakeDeleteRequest(
				t,
				router,
				"/api/v1/databases/"+database.ID.String(),
				"Bearer "+testUserToken,
				tt.expectedStatusCode,
			)

			if !tt.expectSuccess {
				defer RemoveTestDatabase(t.Context(), database)
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_GetDatabase_PermissionsEnforced(t *testing.T) {
	memberRole := users_enums.WorkspaceRoleViewer
	tests := []struct {
		name               string
		userRole           *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace member can get database",
			userRole:           &memberRole,
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot get database",
			userRole:           nil,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can get database",
			userRole:           nil,
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
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(t.Context(), database)

			var testUser string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUser = admin.Token
			} else if tt.userRole != nil {
				member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					member,
					*tt.userRole,
					owner.Token,
					router,
				)
				testUser = member.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				testUser = nonMember.Token
			}

			var response Database
			testResp := test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/"+database.ID.String(),
				"Bearer "+testUser,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.ID)
				assert.Equal(t, "Test Database", response.Name)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_GetDatabasesByWorkspace_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		isMember           bool
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace member can list databases",
			isMember:           true,
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "non-member cannot list databases",
			isMember:           false,
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can list databases",
			isMember:           false,
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
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			db1 := createTestDatabaseViaAPI("Database 1", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(t.Context(), db1)
			db2 := createTestDatabaseViaAPI("Database 2", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(t.Context(), db2)

			var testUser string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUser = admin.Token
			} else if tt.isMember {
				testUser = owner.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				testUser = nonMember.Token
			}

			if tt.expectSuccess {
				var response []Database
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					"/api/v1/databases?workspace_id="+workspace.ID.String(),
					"Bearer "+testUser,
					tt.expectedStatusCode,
					&response,
				)
				assert.GreaterOrEqual(t, len(response), 2)
			} else {
				testResp := test_utils.MakeGetRequest(
					t,
					router,
					"/api/v1/databases?workspace_id="+workspace.ID.String(),
					"Bearer "+testUser,
					tt.expectedStatusCode,
				)
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_GetDatabasesByWorkspace_WhenMultipleDatabasesExist_ReturnsCorrectCount(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	db1 := createTestDatabaseViaAPI("Database 1", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(t.Context(), db1)
	db2 := createTestDatabaseViaAPI("Database 2", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(t.Context(), db2)
	db3 := createTestDatabaseViaAPI("Database 3", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(t.Context(), db3)

	var response []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, 3, len(response))
}

func Test_GetDatabasesByWorkspace_WithPhysicalBackups_ReturnsNewestBackupTimeAcrossTypes(t *testing.T) {
	const walSegmentBytes = 16 * 1024 * 1024

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(
		t.Context(),
		"Phys Last Backup "+uuid.NewString(),
		owner,
		router,
	)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	storage := storages.CreateTestStorage(workspace.ID)
	defer storages.RemoveTestStorage(t.Context(), storage.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	defer notifiers.RemoveTestNotifier(notifier)

	backedUp := CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	defer func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, backedUp.ID)
		RemoveTestDatabase(t.Context(), backedUp)
	}()
	noBackups := CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	defer RemoveTestDatabase(t.Context(), noBackups)

	base := time.Now().UTC().Add(-time.Hour)

	fullBackup := physical_testing.NewTestCompletedFullBackup(
		backedUp.ID, storage.ID, 1, walmath.LSN(0), walmath.LSN(walSegmentBytes))
	fullBackup.CreatedAt = base
	fullBackup.CompletedAt = new(base)
	physical_testing.CreateTestFullBackup(t, fullBackup)

	incrementalBackup := physical_testing.NewTestCompletedIncrementalBackup(
		backedUp.ID, storage.ID, fullBackup.ID, nil, 1, walmath.LSN(walSegmentBytes), walmath.LSN(2*walSegmentBytes))
	incrementalBackup.CreatedAt = base.Add(10 * time.Minute)
	physical_testing.CreateTestIncrementalBackup(t, incrementalBackup)

	// Newest of the three - the value the card must show.
	walSegment := physical_testing.NewTestWalSegment(
		backedUp.ID, storage.ID, 1, "000000010000000000000001",
		walmath.LSN(2*walSegmentBytes), walmath.LSN(3*walSegmentBytes))
	walSegment.ReceivedAt = base.Add(20 * time.Minute)
	physical_testing.CreateTestWalSegment(t, walSegment)

	var response []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	backedUpListed := findDatabaseByID(t, response, backedUp.ID)
	require.NotNil(t, backedUpListed.LastBackupTime, "physical database with backups must expose a last backup time")
	assert.WithinDuration(t, walSegment.ReceivedAt, *backedUpListed.LastBackupTime, time.Second)

	noBackupsListed := findDatabaseByID(t, response, noBackups.ID)
	assert.Nil(t, noBackupsListed.LastBackupTime, "physical database without backups must have no last backup time")
}

func findDatabaseByID(t *testing.T, databases []Database, databaseID uuid.UUID) Database {
	t.Helper()

	for _, database := range databases {
		if database.ID == databaseID {
			return database
		}
	}

	t.Fatalf("database %s not found in response", databaseID)

	return Database{}
}

func Test_GetDatabasesByWorkspace_EnsuresCrossWorkspaceIsolation(t *testing.T) {
	router := createTestRouter()
	owner1 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", owner1, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1, router)

	owner2 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", owner2, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2, router)

	workspace1Db1 := createTestDatabaseViaAPI("Workspace1 DB1", workspace1.ID, owner1.Token, router)
	defer RemoveTestDatabase(t.Context(), workspace1Db1)
	workspace1Db2 := createTestDatabaseViaAPI("Workspace1 DB2", workspace1.ID, owner1.Token, router)
	defer RemoveTestDatabase(t.Context(), workspace1Db2)

	workspace2Db1 := createTestDatabaseViaAPI("Workspace2 DB1", workspace2.ID, owner2.Token, router)
	defer RemoveTestDatabase(t.Context(), workspace2Db1)

	var workspace1Dbs []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace1.ID.String(),
		"Bearer "+owner1.Token,
		http.StatusOK,
		&workspace1Dbs,
	)

	var workspace2Dbs []Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases?workspace_id="+workspace2.ID.String(),
		"Bearer "+owner2.Token,
		http.StatusOK,
		&workspace2Dbs,
	)

	assert.Equal(t, 2, len(workspace1Dbs))
	assert.Equal(t, 1, len(workspace2Dbs))

	for _, db := range workspace1Dbs {
		assert.Equal(t, workspace1.ID, *db.WorkspaceID)
	}

	for _, db := range workspace2Dbs {
		assert.Equal(t, workspace2.ID, *db.WorkspaceID)
	}
}

func Test_CopyDatabase_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can copy database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace member can copy database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "workspace viewer cannot copy database",
			workspaceRole:      func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can copy database",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(t.Context(), database)

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

			var response Database
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/"+database.ID.String()+"/copy",
				"Bearer "+testUserToken,
				nil,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				defer RemoveTestDatabase(t.Context(), &response)
				assert.NotEqual(t, database.ID, response.ID)
				assert.Contains(t, response.Name, "(Copy)")
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_CopyDatabase_CopyStaysInSameWorkspace(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(t.Context(), database)

	var response Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database.ID.String()+"/copy",
		"Bearer "+owner.Token,
		nil,
		http.StatusCreated,
		&response,
	)

	defer RemoveTestDatabase(t.Context(), &response)

	assert.NotEqual(t, database.ID, response.ID)
	assert.Equal(t, "Test Database (Copy)", response.Name)
	assert.Equal(t, workspace.ID, *response.WorkspaceID)
	assert.Equal(t, database.Type, response.Type)
}

func Test_CreateDatabase_PasswordIsEncryptedInDB(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	pgConfig := getTestPostgresConfig()
	plainPassword := "testpassword"
	pgConfig.Password = plainPassword
	request := Database{
		Name:              "Test Database",
		WorkspaceID:       &workspace.ID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: pgConfig,
	}

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+owner.Token,
		request,
		http.StatusCreated,
		&createdDatabase,
	)

	repository := &DatabaseRepository{}
	databaseFromDB, err := repository.FindByID(createdDatabase.ID)
	assert.NoError(t, err)
	assert.NotNil(t, databaseFromDB)
	assert.NotNil(t, databaseFromDB.PostgresqlLogical)

	assert.True(
		t,
		strings.HasPrefix(databaseFromDB.PostgresqlLogical.Password, "enc:"),
		"Password should be encrypted in database with 'enc:' prefix, got: %s",
		databaseFromDB.PostgresqlLogical.Password,
	)

	encryptor := encryption.GetFieldEncryptor()
	decryptedPassword, err := encryptor.Decrypt(databaseFromDB.PostgresqlLogical.Password)
	assert.NoError(t, err)
	assert.Equal(t, plainPassword, decryptedPassword,
		"Decrypted password should match original plaintext password")

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+createdDatabase.ID.String(),
		"Bearer "+owner.Token,
		http.StatusNoContent,
	)

	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

const updatedSshTunnelHost = "moved-bastion.example.com"

// Populated but disabled: Validate skips a disabled tunnel and no code path dials it, while
// HideSensitiveData, Update and EncryptSensitiveFields all run regardless. That is what lets the
// secret lifecycle be asserted over HTTP against a database with no bastion in front of it.
func storedSshTunnelConfig() sshtunnel.Config {
	return sshtunnel.Config{
		IsEnabled:            false,
		Host:                 "bastion.example.com",
		Port:                 2222,
		Username:             "tunneluser",
		AuthType:             sshtunnel.AuthTypePrivateKey,
		PrivateKey:           "tunnelprivatekey",
		PrivateKeyPassphrase: "tunnelpassphrase",
	}
}

// The host moves as well, so an engine that stopped delegating to sshtunnel.Config.Update would
// keep the stored secrets by doing nothing at all and still look correct. The auth type is left out
// alongside the secrets: blank means unchanged for it too, and overwriting it would take the stored
// key down with it.
func submittedSshTunnelConfigWithoutSecrets() sshtunnel.Config {
	sshTunnel := storedSshTunnelConfig()
	sshTunnel.Host = updatedSshTunnelHost
	sshTunnel.AuthType = ""
	sshTunnel.PrivateKey = ""
	sshTunnel.PrivateKeyPassphrase = ""

	return sshTunnel
}

func assertSshTunnelUpdateMovedTheHostAndKeptTheSecrets(t *testing.T, sshTunnel sshtunnel.Config) {
	t.Helper()

	assert.Equal(t, updatedSshTunnelHost, sshTunnel.Host,
		"the address is always submitted, so the update must land")
	assert.Equal(t, sshtunnel.AuthTypePrivateKey, sshTunnel.AuthType,
		"a blank auth type must keep the stored one, or the secrets below are cleared with it")

	encryptor := encryption.GetFieldEncryptor()

	for submittedSecret, storedSecret := range map[string]string{
		"tunnelprivatekey": sshTunnel.PrivateKey,
		"tunnelpassphrase": sshTunnel.PrivateKeyPassphrase,
	} {
		assert.True(t, strings.HasPrefix(storedSecret, "enc:"),
			"SSH tunnel secret should be encrypted in database")

		decrypted, err := encryptor.Decrypt(storedSecret)
		assert.NoError(t, err)
		assert.Equal(t, submittedSecret, decrypted)
	}
}

func assertSshTunnelSecretsAreHidden(t *testing.T, sshTunnel sshtunnel.Config) {
	t.Helper()

	assert.Empty(t, sshTunnel.Password)
	assert.Empty(t, sshTunnel.PrivateKey)
	assert.Empty(t, sshTunnel.PrivateKeyPassphrase)
	assert.Equal(t, "tunneluser", sshTunnel.Username, "the address stays readable in the UI")
}

func Test_DatabaseSensitiveDataLifecycle_AllTypes(t *testing.T) {
	// Started once and shared by the physical case's create and update: system_identifier is
	// immutable, so pointing the update at a second cluster is refused as a cluster swap.
	physicalSource := containers.StartPhysicalPostgres(t, "postgres:17")

	testCases := []struct {
		name                string
		databaseType        DatabaseType
		createDatabase      func(t *testing.T, workspaceID uuid.UUID) *Database
		updateDatabase      func(t *testing.T, workspaceID, databaseID uuid.UUID) *Database
		verifySensitiveData func(t *testing.T, database *Database)
		verifyHiddenData    func(t *testing.T, database *Database)
	}{
		{
			name:         "PostgreSQL Database",
			databaseType: DatabaseTypePostgresLogical,
			createDatabase: func(_ *testing.T, workspaceID uuid.UUID) *Database {
				pgConfig := getTestPostgresConfig()
				pgConfig.SshTunnel = storedSshTunnelConfig()
				return &Database{
					WorkspaceID:       &workspaceID,
					Name:              "Test PostgreSQL Database",
					Type:              DatabaseTypePostgresLogical,
					PostgresqlLogical: pgConfig,
				}
			},
			updateDatabase: func(_ *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				pgConfig := getTestPostgresConfig()
				pgConfig.Password = ""
				pgConfig.SshTunnel = submittedSshTunnelConfigWithoutSecrets()
				return &Database{
					ID:                databaseID,
					WorkspaceID:       &workspaceID,
					Name:              "Updated PostgreSQL Database",
					Type:              DatabaseTypePostgresLogical,
					PostgresqlLogical: pgConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.PostgresqlLogical.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.PostgresqlLogical.Password)
				assert.NoError(t, err)
				assert.Equal(t, "testpassword", decrypted)
				assertSshTunnelUpdateMovedTheHostAndKeptTheSecrets(t, database.PostgresqlLogical.SshTunnel)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.PostgresqlLogical.Password)
				assertSshTunnelSecretsAreHidden(t, database.PostgresqlLogical.SshTunnel)
			},
		},
		{
			name:         "PostgreSQL Physical Database",
			databaseType: DatabaseTypePostgresPhysical,
			createDatabase: func(_ *testing.T, workspaceID uuid.UUID) *Database {
				physicalConfig := getTestPhysicalConfigForSource(physicalSource)
				physicalConfig.SshTunnel = storedSshTunnelConfig()
				return &Database{
					WorkspaceID:        &workspaceID,
					Name:               "Test PostgreSQL Physical Database",
					Type:               DatabaseTypePostgresPhysical,
					PostgresqlPhysical: physicalConfig,
				}
			},
			updateDatabase: func(_ *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				physicalConfig := getTestPhysicalConfigForSource(physicalSource)
				physicalConfig.Password = ""
				physicalConfig.SshTunnel = submittedSshTunnelConfigWithoutSecrets()
				return &Database{
					ID:                 databaseID,
					WorkspaceID:        &workspaceID,
					Name:               "Updated PostgreSQL Physical Database",
					Type:               DatabaseTypePostgresPhysical,
					PostgresqlPhysical: physicalConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.PostgresqlPhysical.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.PostgresqlPhysical.Password)
				assert.NoError(t, err)
				assert.Equal(t, "testpassword", decrypted)
				assertSshTunnelUpdateMovedTheHostAndKeptTheSecrets(t, database.PostgresqlPhysical.SshTunnel)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.PostgresqlPhysical.Password)
				assertSshTunnelSecretsAreHidden(t, database.PostgresqlPhysical.SshTunnel)
			},
		},
		{
			name:         "MariaDB Database",
			databaseType: DatabaseTypeMariadb,
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) *Database {
				mariaConfig := getTestMariadbConfig(t)
				mariaConfig.SshTunnel = storedSshTunnelConfig()
				return &Database{
					WorkspaceID: &workspaceID,
					Name:        "Test MariaDB Database",
					Type:        DatabaseTypeMariadb,
					Mariadb:     mariaConfig,
				}
			},
			updateDatabase: func(t *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				mariaConfig := getTestMariadbConfig(t)
				mariaConfig.Password = ""
				mariaConfig.SshTunnel = submittedSshTunnelConfigWithoutSecrets()
				return &Database{
					ID:          databaseID,
					WorkspaceID: &workspaceID,
					Name:        "Updated MariaDB Database",
					Type:        DatabaseTypeMariadb,
					Mariadb:     mariaConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.Mariadb.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.Mariadb.Password)
				assert.NoError(t, err)
				assert.Equal(t, "testpassword", decrypted)
				assertSshTunnelUpdateMovedTheHostAndKeptTheSecrets(t, database.Mariadb.SshTunnel)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.Mariadb.Password)
				assertSshTunnelSecretsAreHidden(t, database.Mariadb.SshTunnel)
			},
		},
		{
			name:         "MySQL Database",
			databaseType: DatabaseTypeMysql,
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) *Database {
				mysqlConfig := getTestMysqlConfig(t)
				mysqlConfig.SshTunnel = storedSshTunnelConfig()
				return &Database{
					WorkspaceID: &workspaceID,
					Name:        "Test MySQL Database",
					Type:        DatabaseTypeMysql,
					Mysql:       mysqlConfig,
				}
			},
			updateDatabase: func(t *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				mysqlConfig := getTestMysqlConfig(t)
				mysqlConfig.Password = ""
				mysqlConfig.SshTunnel = submittedSshTunnelConfigWithoutSecrets()
				return &Database{
					ID:          databaseID,
					WorkspaceID: &workspaceID,
					Name:        "Updated MySQL Database",
					Type:        DatabaseTypeMysql,
					Mysql:       mysqlConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.Mysql.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.Mysql.Password)
				assert.NoError(t, err)
				assert.Equal(t, "testpassword", decrypted)
				assertSshTunnelUpdateMovedTheHostAndKeptTheSecrets(t, database.Mysql.SshTunnel)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.Mysql.Password)
				assertSshTunnelSecretsAreHidden(t, database.Mysql.SshTunnel)
			},
		},
		{
			name:         "MongoDB Database",
			databaseType: DatabaseTypeMongodb,
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) *Database {
				mongoConfig := getTestMongodbConfig(t)
				mongoConfig.SshTunnel = storedSshTunnelConfig()
				return &Database{
					WorkspaceID: &workspaceID,
					Name:        "Test MongoDB Database",
					Type:        DatabaseTypeMongodb,
					Mongodb:     mongoConfig,
				}
			},
			updateDatabase: func(t *testing.T, workspaceID, databaseID uuid.UUID) *Database {
				mongoConfig := getTestMongodbConfig(t)
				mongoConfig.Password = ""
				mongoConfig.SshTunnel = submittedSshTunnelConfigWithoutSecrets()
				return &Database{
					ID:          databaseID,
					WorkspaceID: &workspaceID,
					Name:        "Updated MongoDB Database",
					Type:        DatabaseTypeMongodb,
					Mongodb:     mongoConfig,
				}
			},
			verifySensitiveData: func(t *testing.T, database *Database) {
				assert.True(t, strings.HasPrefix(database.Mongodb.Password, "enc:"),
					"Password should be encrypted in database")

				encryptor := encryption.GetFieldEncryptor()
				decrypted, err := encryptor.Decrypt(database.Mongodb.Password)
				assert.NoError(t, err)
				assert.Equal(t, "rootpassword", decrypted)
				assertSshTunnelUpdateMovedTheHostAndKeptTheSecrets(t, database.Mongodb.SshTunnel)
			},
			verifyHiddenData: func(t *testing.T, database *Database) {
				assert.Equal(t, "", database.Mongodb.Password)
				assertSshTunnelSecretsAreHidden(t, database.Mongodb.SshTunnel)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			// Phase 1: Create database with sensitive data
			initialDatabase := tc.createDatabase(t, workspace.ID)
			var createdDatabase Database
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/create",
				"Bearer "+owner.Token,
				*initialDatabase,
				http.StatusCreated,
				&createdDatabase,
			)
			assert.NotEmpty(t, createdDatabase.ID)
			assert.Equal(t, initialDatabase.Name, createdDatabase.Name)

			// Phase 2: Read via service - sensitive data should be hidden
			var retrievedDatabase Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/databases/%s", createdDatabase.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&retrievedDatabase,
			)
			tc.verifyHiddenData(t, &retrievedDatabase)
			assert.Equal(t, initialDatabase.Name, retrievedDatabase.Name)

			// Phase 3: Update with non-sensitive changes only (sensitive fields empty)
			updatedDatabase := tc.updateDatabase(t, workspace.ID, createdDatabase.ID)
			var updateResponse Database
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/update",
				"Bearer "+owner.Token,
				*updatedDatabase,
				http.StatusOK,
				&updateResponse,
			)

			// Phase 4: Retrieve directly from repository to verify sensitive data preservation
			repository := &DatabaseRepository{}
			databaseFromDB, err := repository.FindByID(createdDatabase.ID)
			assert.NoError(t, err)

			// Verify original sensitive data is still present in DB
			tc.verifySensitiveData(t, databaseFromDB)

			// Verify non-sensitive fields were updated in DB
			assert.Equal(t, updatedDatabase.Name, databaseFromDB.Name)

			// Phase 5: Additional verification - Check via GET that data is still hidden
			var finalRetrieved Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/databases/%s", createdDatabase.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&finalRetrieved,
			)
			tc.verifyHiddenData(t, &finalRetrieved)

			// Phase 6: Verify GetDatabasesByWorkspace also hides sensitive data
			var workspaceDatabases []Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/databases?workspace_id=%s", workspace.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&workspaceDatabases,
			)
			var foundDatabase *Database
			for i := range workspaceDatabases {
				if workspaceDatabases[i].ID == createdDatabase.ID {
					foundDatabase = &workspaceDatabases[i]
					break
				}
			}
			assert.NotNil(t, foundDatabase, "Database should be found in workspace databases list")
			tc.verifyHiddenData(t, foundDatabase)

			// Clean up: Delete database before removing workspace
			test_utils.MakeDeleteRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/databases/%s", createdDatabase.ID.String()),
				"Bearer "+owner.Token,
				http.StatusNoContent,
			)

			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_TestConnection_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name                    string
		isMember                bool
		isGlobalAdmin           bool
		expectAccessGranted     bool
		expectedStatusCodeOnErr int
	}{
		{
			name:                    "workspace member can test connection",
			isMember:                true,
			isGlobalAdmin:           false,
			expectAccessGranted:     true,
			expectedStatusCodeOnErr: http.StatusBadRequest,
		},
		{
			name:                    "non-member cannot test connection",
			isMember:                false,
			isGlobalAdmin:           false,
			expectAccessGranted:     false,
			expectedStatusCodeOnErr: http.StatusBadRequest,
		},
		{
			name:                    "global admin can test connection",
			isMember:                false,
			isGlobalAdmin:           true,
			expectAccessGranted:     true,
			expectedStatusCodeOnErr: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
			defer RemoveTestDatabase(t.Context(), database)

			var testUser string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUser = admin.Token
			} else if tt.isMember {
				testUser = owner.Token
			} else {
				nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				testUser = nonMember.Token
			}

			w := workspaces_testing.MakeAPIRequest(
				router,
				"POST",
				"/api/v1/databases/"+database.ID.String()+"/test-connection",
				"Bearer "+testUser,
				nil,
			)

			body := w.Body.String()

			if tt.expectAccessGranted {
				assert.True(
					t,
					w.Code == http.StatusOK ||
						(w.Code == http.StatusBadRequest && strings.Contains(body, "connect")),
					"Expected 200 OK or 400 with connection error, got %d: %s",
					w.Code,
					body,
				)
			} else {
				assert.Equal(t, tt.expectedStatusCodeOnErr, w.Code)
				assert.Contains(t, body, "insufficient permissions")
			}
		})
	}
}

func Test_UpdateDatabase_WhenExcludeTablesArePastedMultiline_StoresTrimmedUniqueNames(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	defer RemoveTestDatabase(t.Context(), database)

	database.PostgresqlLogical.ExcludeTables = []string{"orders", "\npersonnel_real_time", " ", "orders"}
	database.PostgresqlLogical.IncludeSchemas = []string{"public", "\treporting"}
	database.PostgresqlLogical.ExcludeTableData = []string{"logs, history_*", "\nlogs", " "}

	var updatedDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database,
		http.StatusOK,
		&updatedDatabase,
	)

	var reloadedDatabase Database
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&reloadedDatabase,
	)

	assert.Equal(
		t,
		[]string{"orders", "personnel_real_time"},
		reloadedDatabase.PostgresqlLogical.ExcludeTables,
	)
	assert.Equal(
		t,
		[]string{"public", "reporting"},
		reloadedDatabase.PostgresqlLogical.IncludeSchemas,
	)
	assert.Equal(
		t,
		[]string{"logs", "history_*"},
		reloadedDatabase.PostgresqlLogical.ExcludeTableData,
	)
}

func Test_ExcludeTableData_WhenDatabaseIsCreatedCopiedAndUpdatedWithoutIt_ListIsKeptCopiedAndCleared(
	t *testing.T,
) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	createRequest := newTestPostgresqlLogicalDatabaseRequest("Data-excluded Database", workspace.ID)
	createRequest.PostgresqlLogical.ExcludeTables = []string{"audit_tmp"}
	createRequest.PostgresqlLogical.ExcludeTableData = []string{"logs", "history_*"}

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/create", "Bearer "+owner.Token,
		createRequest, http.StatusCreated, &createdDatabase,
	)
	defer RemoveTestDatabase(t.Context(), &createdDatabase)

	var readDatabase Database
	test_utils.MakeGetRequestAndUnmarshal(
		t, router, "/api/v1/databases/"+createdDatabase.ID.String(),
		"Bearer "+owner.Token, http.StatusOK, &readDatabase,
	)
	require.NotNil(t, readDatabase.PostgresqlLogical)
	assert.Equal(t, []string{"audit_tmp"}, readDatabase.PostgresqlLogical.ExcludeTables)
	assert.Equal(t, []string{"logs", "history_*"}, readDatabase.PostgresqlLogical.ExcludeTableData)

	var copiedDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/"+createdDatabase.ID.String()+"/copy",
		"Bearer "+owner.Token, nil, http.StatusCreated, &copiedDatabase,
	)
	defer RemoveTestDatabase(t.Context(), &copiedDatabase)

	var readCopiedDatabase Database
	test_utils.MakeGetRequestAndUnmarshal(
		t, router, "/api/v1/databases/"+copiedDatabase.ID.String(),
		"Bearer "+owner.Token, http.StatusOK, &readCopiedDatabase,
	)
	require.NotNil(t, readCopiedDatabase.PostgresqlLogical)
	assert.Equal(
		t, []string{"logs", "history_*"}, readCopiedDatabase.PostgresqlLogical.ExcludeTableData,
	)

	readDatabase.PostgresqlLogical.ExcludeTableData = nil

	var updatedDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/update", "Bearer "+owner.Token,
		readDatabase, http.StatusOK, &updatedDatabase,
	)

	var reloadedDatabase Database
	test_utils.MakeGetRequestAndUnmarshal(
		t, router, "/api/v1/databases/"+createdDatabase.ID.String(),
		"Bearer "+owner.Token, http.StatusOK, &reloadedDatabase,
	)
	require.NotNil(t, reloadedDatabase.PostgresqlLogical)
	assert.Empty(t, reloadedDatabase.PostgresqlLogical.ExcludeTableData)
	assert.Equal(t, []string{"audit_tmp"}, reloadedDatabase.PostgresqlLogical.ExcludeTables)
}

func createTestDatabaseViaAPI(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *Database {
	request := newTestPostgresqlLogicalDatabaseRequest(name, workspaceID)

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

	var database Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}

	return &database
}

func newTestPostgresqlLogicalDatabaseRequest(name string, workspaceID uuid.UUID) Database {
	return Database{
		Name:              name,
		WorkspaceID:       &workspaceID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}
}

func createTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected.(*gin.RouterGroup))
	workspaces_controllers.GetMembershipController().RegisterRoutes(protected.(*gin.RouterGroup))
	GetDatabaseController().RegisterRoutes(protected.(*gin.RouterGroup))

	GetDatabaseController().RegisterPublicRoutes(v1)

	audit_logs.SetupDependencies()

	return router
}

func getTestPostgresConfig() *postgresql_logical.PostgresqlLogicalDatabase {
	env := config.GetEnv()
	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse TEST_LOGICAL_POSTGRES_16_PORT: %v", err))
	}

	testDbName := "testdb"
	return &postgresql_logical.PostgresqlLogicalDatabase{
		Version:  tools.PostgresqlVersion16,
		Host:     config.GetEnv().TestLocalhost,
		Port:     port,
		Username: "testuser",
		Password: "testpassword",
		Database: &testDbName,
		CpuCount: 1,
	}
}

func Test_CreateDatabase_WhenUserIsNotReadOnly_DatabaseCreated(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Non-Cloud", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	request := Database{
		Name:              "Non-Cloud DB",
		WorkspaceID:       &workspace.ID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}

	var response Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+owner.Token,
		request,
		http.StatusCreated,
		&response,
	)
	defer RemoveTestDatabase(t.Context(), &response)

	assert.Equal(t, "Non-Cloud DB", response.Name)
	assert.NotEqual(t, uuid.Nil, response.ID)
}

func getTestMariadbConfig(t *testing.T) *mariadb.MariadbDatabase {
	endpoint := containers.StartMariadb(t, "mariadb:10.11")

	return GetTestMariadbConfig(endpoint.Host, endpoint.Port)
}

func getTestMysqlConfig(t *testing.T) *mysql.MysqlDatabase {
	endpoint := containers.StartMysql(t, "mysql:8.0")

	return GetTestMysqlConfig(endpoint.Host, endpoint.Port)
}

func getTestPhysicalConfigForSource(source containers.Endpoint) *postgresql_physical.PostgresqlPhysicalDatabase {
	return GetTestPhysicalPostgresConfigWithType(
		source.Host, source.Port, "17", postgresql_physical.BackupTypeFullOnly,
	)
}

func getTestMongodbConfig(t *testing.T) *mongodb.MongodbDatabase {
	endpoint := containers.StartMongodb(t, "mongo:7.0")

	return GetTestMongodbConfig(endpoint.Host, endpoint.Port)
}

// physicalNoSummaryVersion pairs a version tag with its image for the summarize_wal=off tests,
// which boot a throwaway no-summary source per version via containers.StartPhysicalPostgres.
type physicalNoSummaryVersion struct {
	tag   string
	image string
}

var physicalNoSummaryVersions = []physicalNoSummaryVersion{
	{"17", "postgres:17"},
	{"18", "postgres:18"},
}

func Test_CreateDatabase_FailsForPhysicalIncrementalWhenSummarizeWalOff(t *testing.T) {
	incrementalBackupTypes := []postgresql_physical.BackupType{
		postgresql_physical.BackupTypeFullAndIncremental,
		postgresql_physical.BackupTypeFullIncrementalAndWalStream,
	}

	for _, dbVersion := range physicalNoSummaryVersions {
		for _, backupType := range incrementalBackupTypes {
			t.Run(fmt.Sprintf("pg%s_%s", dbVersion.tag, backupType), func(t *testing.T) {
				router := createTestRouter()
				owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
				defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

				source := containers.StartPhysicalPostgres(t, dbVersion.image, containers.WithoutSummarizer())
				physicalConfig := GetTestPhysicalPostgresConfigNoSummary(source.Host, source.Port, dbVersion.tag)
				physicalConfig.BackupType = backupType

				request := Database{
					Name:               "Physical Incremental NoSummary",
					WorkspaceID:        &workspace.ID,
					Type:               DatabaseTypePostgresPhysical,
					PostgresqlPhysical: physicalConfig,
				}

				resp := test_utils.MakePostRequest(
					t,
					router,
					"/api/v1/databases/create",
					"Bearer "+owner.Token,
					request,
					http.StatusBadRequest,
				)

				assert.Contains(t, string(resp.Body), string(postgresql_shared.ConnErrWalSummaryDisabled))
			})
		}
	}
}

func Test_UpdateDatabase_FailsForSwitchToPhysicalIncrementalWhenSummarizeWalOff(t *testing.T) {
	for _, dbVersion := range physicalNoSummaryVersions {
		t.Run("pg"+dbVersion.tag, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			source := containers.StartPhysicalPostgres(t, dbVersion.image, containers.WithoutSummarizer())
			createRequest := Database{
				Name:               "Physical Full Only NoSummary",
				WorkspaceID:        &workspace.ID,
				Type:               DatabaseTypePostgresPhysical,
				PostgresqlPhysical: GetTestPhysicalPostgresConfigNoSummary(source.Host, source.Port, dbVersion.tag),
			}

			var createdDatabase Database
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/create",
				"Bearer "+owner.Token,
				createRequest,
				http.StatusCreated,
				&createdDatabase,
			)
			defer RemoveTestDatabase(t.Context(), &createdDatabase)

			createdDatabase.PostgresqlPhysical.BackupType = postgresql_physical.BackupTypeFullAndIncremental

			updateResponse := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/databases/update",
				"Bearer "+owner.Token,
				createdDatabase,
				http.StatusBadRequest,
			)

			assert.Contains(t, string(updateResponse.Body), string(postgresql_shared.ConnErrWalSummaryDisabled))

			var refetchedDatabase Database
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				"/api/v1/databases/"+createdDatabase.ID.String(),
				"Bearer "+owner.Token,
				http.StatusOK,
				&refetchedDatabase,
			)

			assert.Equal(
				t,
				postgresql_physical.BackupTypeFullOnly,
				refetchedDatabase.PostgresqlPhysical.BackupType,
			)
		})
	}
}

func bastionedPostgresConfig(
	topology containers.BastionedDatabase,
) *postgresql_logical.PostgresqlLogicalDatabase {
	databaseName := containers.PostgresDatabase

	return &postgresql_logical.PostgresqlLogicalDatabase{
		Host:     topology.Database.Host,
		Port:     topology.Database.Port,
		Username: containers.PostgresUsername,
		Password: containers.PostgresPassword,
		Database: &databaseName,
		CpuCount: 1,
		SshTunnel: sshtunnel.Config{
			IsEnabled: true,
			Host:      topology.Bastion.Host,
			Port:      topology.Bastion.Port,
			Username:  containers.SshBastionUsername,
			AuthType:  sshtunnel.AuthTypePassword,
			Password:  containers.SshBastionPassword,
		},
	}
}

func readBastionTestKey(t *testing.T) string {
	t.Helper()

	privateKey, err := os.ReadFile(filepath.Join(containers.GetSshBastionTestdataDir(t), "test_key"))
	require.NoError(t, err)

	return string(privateKey)
}

// Without this the tunnel tests would stay green if the tunnel silently stopped being used, because
// every other assertion only proves that some route to the database exists.
func Test_BastionedPostgres_WithoutTheTunnel_IsUnreachableFromTheHost(t *testing.T) {
	topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")

	address := net.JoinHostPort(topology.Database.Host, strconv.Itoa(topology.Database.Port))

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err == nil {
		_ = conn.Close()
		t.Fatalf("the bastioned database answered on %s without a tunnel", address)
	}
}

func Test_CreateDatabase_OverSshTunnel_DetectsVersionAndHidesTunnelSecrets(t *testing.T) {
	topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:              "Bastioned PG",
			WorkspaceID:       &workspace.ID,
			Type:              DatabaseTypePostgresLogical,
			PostgresqlLogical: bastionedPostgresConfig(topology),
		},
		http.StatusCreated, &createdDatabase,
	)
	t.Cleanup(func() { RemoveTestDatabase(t.Context(), &createdDatabase) })

	require.NotNil(t, createdDatabase.PostgresqlLogical)
	assert.Equal(t, "16", string(createdDatabase.PostgresqlLogical.Version))

	// The stored config must still address the database as the bastion sees it, not as the
	// forwarder rewrote it for the duration of the connection test.
	assert.Equal(t, topology.Database.Host, createdDatabase.PostgresqlLogical.Host)
	assert.Equal(t, topology.Database.Port, createdDatabase.PostgresqlLogical.Port)
	assert.True(t, createdDatabase.PostgresqlLogical.SshTunnel.IsEnabled)

	var fetchedDatabase Database
	test_utils.MakeGetRequestAndUnmarshal(
		t, router, "/api/v1/databases/"+createdDatabase.ID.String(),
		"Bearer "+owner.Token, http.StatusOK, &fetchedDatabase,
	)

	require.NotNil(t, fetchedDatabase.PostgresqlLogical)
	assert.Empty(t, fetchedDatabase.PostgresqlLogical.SshTunnel.Password)
	assert.Empty(t, fetchedDatabase.PostgresqlLogical.SshTunnel.PrivateKey)
	assert.Empty(t, fetchedDatabase.PostgresqlLogical.SshTunnel.PrivateKeyPassphrase)
	assert.Equal(t, containers.SshBastionUsername, fetchedDatabase.PostgresqlLogical.SshTunnel.Username)

	testConnectionResponse := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/"+createdDatabase.ID.String()+"/test-connection",
		"Bearer "+owner.Token, nil,
	)
	assert.Equal(t, http.StatusOK, testConnectionResponse.Code, "body: %s", testConnectionResponse.Body.String())
}

func Test_CreateDatabase_OverSshTunnelWithAPrivateKey_DatabaseCreated(t *testing.T) {
	topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Key", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	postgresConfig := bastionedPostgresConfig(topology)
	postgresConfig.SshTunnel.AuthType = sshtunnel.AuthTypePrivateKey
	postgresConfig.SshTunnel.Password = ""
	postgresConfig.SshTunnel.PrivateKey = readBastionTestKey(t)

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:              "Bastioned PG by key",
			WorkspaceID:       &workspace.ID,
			Type:              DatabaseTypePostgresLogical,
			PostgresqlLogical: postgresConfig,
		},
		http.StatusCreated, &createdDatabase,
	)
	t.Cleanup(func() { RemoveTestDatabase(t.Context(), &createdDatabase) })

	require.NotNil(t, createdDatabase.PostgresqlLogical)
	assert.Equal(t, "16", string(createdDatabase.PostgresqlLogical.Version))
}

// The stored key would otherwise stay a working way into the bastion after the user replaced it
// with a password.
func Test_UpdateDatabase_WhenSshAuthTypeChangesToPassword_ClearsTheStoredPrivateKey(t *testing.T) {
	topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Auth Switch", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	postgresConfig := bastionedPostgresConfig(topology)
	postgresConfig.SshTunnel.AuthType = sshtunnel.AuthTypePrivateKey
	postgresConfig.SshTunnel.Password = ""
	postgresConfig.SshTunnel.PrivateKey = readBastionTestKey(t)

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:              "Bastioned PG switching auth",
			WorkspaceID:       &workspace.ID,
			Type:              DatabaseTypePostgresLogical,
			PostgresqlLogical: postgresConfig,
		},
		http.StatusCreated, &createdDatabase,
	)
	t.Cleanup(func() { RemoveTestDatabase(t.Context(), &createdDatabase) })

	createdDatabase.PostgresqlLogical.SshTunnel.AuthType = sshtunnel.AuthTypePassword
	createdDatabase.PostgresqlLogical.SshTunnel.Password = containers.SshBastionPassword

	var updatedDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/update", "Bearer "+owner.Token,
		createdDatabase, http.StatusOK, &updatedDatabase,
	)

	persistedDatabase, err := getDatabaseRepository().FindByID(createdDatabase.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedDatabase.PostgresqlLogical)

	assert.Empty(t, persistedDatabase.PostgresqlLogical.SshTunnel.PrivateKey)
	assert.NotEmpty(t, persistedDatabase.PostgresqlLogical.SshTunnel.Password)
}

func Test_CreateDatabase_WhenSshAuthTypeIsPrivateKeyButOnlyAPasswordIsSet_ReturnsBadRequest(
	t *testing.T,
) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Auth Mismatch", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	postgresConfig := getTestPostgresConfig()
	postgresConfig.SshTunnel = sshtunnel.Config{
		IsEnabled: true,
		Host:      "bastion.example.com",
		Port:      22,
		Username:  "tunneluser",
		AuthType:  sshtunnel.AuthTypePrivateKey,
		Password:  "tunnelpassword",
	}

	response := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:              "Tunnel without a key",
			WorkspaceID:       &workspace.ID,
			Type:              DatabaseTypePostgresLogical,
			PostgresqlLogical: postgresConfig,
		},
	)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "SSH tunnel private key is required")
}

// Storing the unused secret would leave a second, invisible way into the bastion behind: the edit
// form only ever shows the chosen one, so nothing would surface it again.
func Test_CreateDatabase_WhenTheSshTunnelCarriesBothSecrets_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Both Secrets", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	postgresConfig := getTestPostgresConfig()
	postgresConfig.SshTunnel = sshtunnel.Config{
		IsEnabled:  true,
		Host:       "bastion.example.com",
		Port:       22,
		Username:   "tunneluser",
		AuthType:   sshtunnel.AuthTypePassword,
		Password:   "tunnelpassword",
		PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----",
	}

	response := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:              "Tunnel with both secrets",
			WorkspaceID:       &workspace.ID,
			Type:              DatabaseTypePostgresLogical,
			PostgresqlLogical: postgresConfig,
		},
	)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "must not carry a private key")
}

func Test_CreateDatabase_WhenSshTunnelIsEnabledWithoutAHost_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Invalid", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	postgresConfig := getTestPostgresConfig()
	postgresConfig.SshTunnel = sshtunnel.Config{
		IsEnabled: true,
		Port:      22,
		Username:  "tunneluser",
		AuthType:  sshtunnel.AuthTypePassword,
		Password:  "tunnelpassword",
	}

	response := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:              "Tunnel without a host",
			WorkspaceID:       &workspace.ID,
			Type:              DatabaseTypePostgresLogical,
			PostgresqlLogical: postgresConfig,
		},
	)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "SSH tunnel host is required")
}

func Test_CopyDatabase_WhenTheSourceIsBehindABastion_KeepsTheTunnelConfig(t *testing.T) {
	topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "SSH Tunnel Copy", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:              "Bastioned PG to copy",
			WorkspaceID:       &workspace.ID,
			Type:              DatabaseTypePostgresLogical,
			PostgresqlLogical: bastionedPostgresConfig(topology),
		},
		http.StatusCreated, &createdDatabase,
	)
	t.Cleanup(func() { RemoveTestDatabase(t.Context(), &createdDatabase) })

	var copiedDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/"+createdDatabase.ID.String()+"/copy",
		"Bearer "+owner.Token, nil, http.StatusCreated, &copiedDatabase,
	)
	t.Cleanup(func() { RemoveTestDatabase(t.Context(), &copiedDatabase) })

	require.NotNil(t, copiedDatabase.PostgresqlLogical)
	assert.True(t, copiedDatabase.PostgresqlLogical.SshTunnel.IsEnabled)
	assert.Equal(t, topology.Bastion.Host, copiedDatabase.PostgresqlLogical.SshTunnel.Host)
	assert.Equal(t, topology.Bastion.Port, copiedDatabase.PostgresqlLogical.SshTunnel.Port)
	assert.Equal(t, containers.SshBastionUsername, copiedDatabase.PostgresqlLogical.SshTunnel.Username)
}

func Test_CreateMongodbDatabase_WhenSrvAndSshTunnelAreEnabled_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Mongo SRV Tunnel", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	response := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create", "Bearer "+owner.Token,
		Database{
			Name:        "SRV behind a bastion",
			WorkspaceID: &workspace.ID,
			Type:        DatabaseTypeMongodb,
			Mongodb: &mongodb.MongodbDatabase{
				Host:         "cluster0.example.mongodb.net",
				Username:     "testuser",
				Password:     "testpassword",
				Database:     "testdb",
				AuthDatabase: "admin",
				CpuCount:     1,
				IsSrv:        true,
				SshTunnel: sshtunnel.Config{
					IsEnabled: true,
					Host:      "bastion.example.com",
					Port:      22,
					Username:  containers.SshBastionUsername,
					AuthType:  sshtunnel.AuthTypePassword,
					Password:  containers.SshBastionPassword,
				},
			},
		},
	)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "SRV")
}

// The source cluster publishes no port, so a green result is proof the request went through the
// bastion — this endpoint is the one physical-only path that had no tunnel around it.
func Test_CreateReplicationOnlyUser_WhenTheDatabaseIsBehindABastion_ReachesTheClusterThroughTheTunnel(
	t *testing.T,
) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Bastioned Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	topology := containers.StartPhysicalPostgresBehindSshBastion(t, "postgres:17")

	physicalConfig := GetTestPhysicalPostgresConfigWithType(
		topology.Database.Host, topology.Database.Port, "17", postgresql_physical.BackupTypeFullOnly,
	)
	physicalConfig.SshTunnel = bastion.GetTunnelConfig(topology)

	request := Database{
		Name:               "Bastioned Physical DB",
		WorkspaceID:        &workspace.ID,
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: physicalConfig,
	}

	var response CreateReadOnlyUserResponse
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create-replication-only-user",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.NotEmpty(t, response.Username)
	assert.NotEmpty(t, response.Password)
}

// Creating a physical database runs the replication-protocol probe, which is the one connection that
// does not go through pgx and so needed its own hostaddr handling. The cluster publishes no port, so
// reaching it at all proves the probe went through the bastion.
func Test_CreateDatabase_WhenThePhysicalClusterIsBehindABastion_PassesTheReplicationConnectionTest(
	t *testing.T,
) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Bastioned Create", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	topology := containers.StartPhysicalPostgresBehindSshBastion(t, "postgres:17")

	physicalConfig := GetTestPhysicalPostgresConfigWithType(
		topology.Database.Host, topology.Database.Port, "17", postgresql_physical.BackupTypeFullOnly,
	)
	physicalConfig.SshTunnel = bastion.GetTunnelConfig(topology)

	request := Database{
		Name:               "Bastioned Physical Create",
		WorkspaceID:        &workspace.ID,
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: physicalConfig,
	}

	var response Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+owner.Token,
		request,
		http.StatusCreated,
		&response,
	)
	defer RemoveTestDatabase(t.Context(), &response)

	require.NotNil(t, response.PostgresqlPhysical)
	assert.NotEmpty(t, response.PostgresqlPhysical.Version,
		"the version discovered through the tunnel must be carried back onto the stored row")
	assert.NotNil(t, response.PostgresqlPhysical.SystemIdentifier,
		"the cluster identity must be carried back too, or the manifest records a zero identifier")
	assert.NotNil(t, response.PostgresqlPhysical.WalSegmentSizeBytes)
}

func openFixtureConn(t *testing.T, fixture physicalFixture) *pgx.Conn {
	t.Helper()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=testuser password=testpassword dbname=postgres sslmode=disable",
		config.GetEnv().TestLocalhost, fixture.port(),
	)

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	return conn
}

func createReplicationRole(t *testing.T, conn *pgx.Conn, canForceWalRotation bool) (string, string) {
	t.Helper()

	username := "databasus-ct-" + uuid.New().String()[:8]
	password := "Temp1234!Pwd"

	_, err := conn.Exec(t.Context(),
		fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN REPLICATION`, username, password))
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	})

	if canForceWalRotation {
		_, err := conn.Exec(t.Context(),
			fmt.Sprintf(`GRANT EXECUTE ON FUNCTION pg_switch_wal() TO "%s"`, username))
		require.NoError(t, err)
	}

	return username, password
}

type physicalDatabaseRequestSpec struct {
	workspaceID uuid.UUID
	fixture     physicalFixture
	username    string
	password    string
	backupType  postgresql_physical.BackupType
}

func buildPhysicalDatabaseRequestFor(t *testing.T, spec physicalDatabaseRequestSpec) Database {
	t.Helper()

	request := buildPhysicalDatabaseRequest(t, spec.workspaceID, spec.fixture)
	request.PostgresqlPhysical.Username = spec.username
	request.PostgresqlPhysical.Password = spec.password
	request.PostgresqlPhysical.BackupType = spec.backupType

	return request
}

func getConnectionErrorCode(t *testing.T, body []byte) postgresql_shared.ConnectionErrorCode {
	t.Helper()

	var payload struct {
		Code postgresql_shared.ConnectionErrorCode `json:"code"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))

	return payload.Code
}

func createDatabaseForDirectConnectionTest(
	t *testing.T,
	router *gin.Engine,
	workspaceID uuid.UUID,
	ownerToken string,
) *Database {
	t.Helper()

	request := Database{
		Name:              "Direct connection test database",
		WorkspaceID:       &workspaceID,
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/create",
		"Bearer "+ownerToken,
		request,
		http.StatusCreated,
		&createdDatabase,
	)

	t.Cleanup(func() {
		RemoveTestDatabase(t.Context(), &createdDatabase)
	})

	return &createdDatabase
}

type connectionTestDatabaseStore struct {
	databaseStore
	database  *Database
	findError error
}

func (s *connectionTestDatabaseStore) FindByID(_ uuid.UUID) (*Database, error) {
	return s.database, s.findError
}

type connectionTestWorkspaceService struct {
	workspaceService
	permissionError error
}

func (s *connectionTestWorkspaceService) CanUserManageDBs(
	_ context.Context,
	_ uuid.UUID,
	_ *users_models.User,
) (bool, error) {
	return false, s.permissionError
}

func createDirectConnectionErrorTestRouter(
	databaseStore databaseStore,
	workspaceService workspaceService,
	user *users_models.User,
) *gin.Engine {
	service := &DatabaseService{
		databaseStore,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		workspaceService,
		nil,
		nil,
		nil,
	}
	controller := &DatabaseController{service, nil, nil}
	router := gin.New()
	router.POST("/api/v1/databases/test-connection-direct", func(ctx *gin.Context) {
		ctx.Set("user", user)
		controller.TestDatabaseConnectionDirect(ctx)
	})

	return router
}

func Test_TestDatabaseConnectionDirect_WhenSavedDatabaseLookupFails_ReturnsSanitizedError(t *testing.T) {
	user := &users_models.User{ID: uuid.New(), Role: users_enums.UserRoleMember}
	router := createDirectConnectionErrorTestRouter(
		&connectionTestDatabaseStore{findError: errors.New("sensitive repository failure")},
		&connectionTestWorkspaceService{},
		user,
	)
	request := Database{ID: uuid.New()}

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/test-connection-direct",
		"",
		request,
		http.StatusInternalServerError,
	)

	assert.JSONEq(t, `{"error":"failed to load database connection target"}`, string(response.Body))
}

func Test_TestDatabaseConnectionDirect_WhenPermissionLookupFails_ReturnsSanitizedError(t *testing.T) {
	workspaceID := uuid.New()
	user := &users_models.User{ID: uuid.New(), Role: users_enums.UserRoleMember}
	permissionError := errors.New("sensitive membership failure")
	testCases := []struct {
		name          string
		databaseStore databaseStore
		request       Database
	}{
		{
			name:          "ad hoc database",
			databaseStore: &connectionTestDatabaseStore{},
			request:       Database{WorkspaceID: &workspaceID},
		},
		{
			name: "saved database",
			databaseStore: &connectionTestDatabaseStore{database: &Database{
				ID:          uuid.New(),
				WorkspaceID: &workspaceID,
			}},
			request: Database{ID: uuid.New()},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := createDirectConnectionErrorTestRouter(
				testCase.databaseStore,
				&connectionTestWorkspaceService{permissionError: permissionError},
				user,
			)

			response := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/databases/test-connection-direct",
				"",
				testCase.request,
				http.StatusInternalServerError,
			)

			assert.JSONEq(t,
				`{"error":"failed to verify database connection permissions"}`,
				string(response.Body),
			)
		})
	}
}

func Test_TestDatabaseConnectionDirect_WithoutWorkspace_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	request := Database{
		Name:              "Direct connection test database",
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/test-connection-direct",
		"Bearer "+user.Token,
		request,
		http.StatusBadRequest,
	)

	assert.JSONEq(t, `{"error":"workspaceId is required"}`, string(response.Body))
}

func Test_TestDatabaseConnectionDirect_ForUnauthorizedSavedDatabase_ReturnsForbidden(t *testing.T) {
	testCases := []struct {
		name                 string
		shouldAddToWorkspace bool
		unauthorizedRole     users_enums.WorkspaceRole
	}{
		{
			name:                 "workspace viewer",
			shouldAddToWorkspace: true,
			unauthorizedRole:     users_enums.WorkspaceRoleViewer,
		},
		{
			name:                 "nonmember",
			shouldAddToWorkspace: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Direct Test Workspace", owner, router)
			t.Cleanup(func() {
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			})

			createdDatabase := createDatabaseForDirectConnectionTest(t, router, workspace.ID, owner.Token)
			unauthorizedUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			if testCase.shouldAddToWorkspace {
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					unauthorizedUser,
					testCase.unauthorizedRole,
					owner.Token,
					router,
				)
			}

			createdDatabase.PostgresqlLogical.Host = "127.0.0.1"
			createdDatabase.PostgresqlLogical.Port = 1
			createdDatabase.PostgresqlLogical.Password = ""

			response := test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/databases/test-connection-direct",
				"Bearer "+unauthorizedUser.Token,
				createdDatabase,
				http.StatusForbidden,
			)

			assert.JSONEq(t,
				`{"error":"insufficient permissions to test this database connection"}`,
				string(response.Body),
			)
		})
	}
}

func Test_TestDatabaseConnectionDirect_ForUnknownDatabase_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	request := Database{
		ID:                uuid.New(),
		Name:              "Unknown database",
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: getTestPostgresConfig(),
	}

	response := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/test-connection-direct",
		"Bearer "+user.Token,
		request,
		http.StatusForbidden,
	)

	assert.JSONEq(t,
		`{"error":"insufficient permissions to test this database connection"}`,
		string(response.Body),
	)
}

func Test_TestDatabaseConnectionDirect_ForWorkspaceMember_ReturnsSuccess(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Direct Test Workspace", owner, router)
	t.Cleanup(func() {
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	})

	createdDatabase := createDatabaseForDirectConnectionTest(t, router, workspace.ID, owner.Token)
	member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		member,
		users_enums.WorkspaceRoleMember,
		owner.Token,
		router,
	)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/test-connection-direct",
		"Bearer "+member.Token,
		createdDatabase,
		http.StatusOK,
	)
}

func Test_TestDatabaseConnectionDirect_ForWalStreamWhenRoleCannotSwitchWal_ReturnsNoWalSwitchPrivilegeCode(
	t *testing.T,
) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			username, password := createReplicationRole(t, openFixtureConn(t, fixture), false)

			request := buildPhysicalDatabaseRequestFor(t, physicalDatabaseRequestSpec{
				workspaceID: workspace.ID,
				fixture:     fixture,
				username:    username,
				password:    password,
				backupType:  postgresql_physical.BackupTypeFullIncrementalAndWalStream,
			})

			connectionTestResponse := test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/test-connection-direct",
				"Bearer "+owner.Token,
				request,
				http.StatusBadRequest,
			)

			assert.Equal(t,
				postgresql_shared.ConnErrNoWalSwitchPrivilege,
				getConnectionErrorCode(t, connectionTestResponse.Body))
		})
	}
}

func Test_TestDatabaseConnectionDirect_ForWalStreamWhenRoleCanSwitchWal_ReturnsSuccess(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			username, password := createReplicationRole(t, openFixtureConn(t, fixture), true)

			request := buildPhysicalDatabaseRequestFor(t, physicalDatabaseRequestSpec{
				workspaceID: workspace.ID,
				fixture:     fixture,
				username:    username,
				password:    password,
				backupType:  postgresql_physical.BackupTypeFullIncrementalAndWalStream,
			})

			test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/test-connection-direct",
				"Bearer "+owner.Token,
				request,
				http.StatusOK,
			)
		})
	}
}

// A saved streaming database whose source revoked the privilege afterwards is the only way
// this state reaches an existing row, since creating it now is refused.
func Test_TestDatabaseConnection_WhenTheSourceRevokedTheWalSwitchPrivilege_ReturnsNoWalSwitchPrivilegeCode(
	t *testing.T,
) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			conn := openFixtureConn(t, fixture)
			username, password := createReplicationRole(t, conn, true)

			request := buildPhysicalDatabaseRequestFor(t, physicalDatabaseRequestSpec{
				workspaceID: workspace.ID,
				fixture:     fixture,
				username:    username,
				password:    password,
				backupType:  postgresql_physical.BackupTypeFullIncrementalAndWalStream,
			})

			createDatabaseResponse := workspaces_testing.MakeAPIRequest(
				router, "POST", "/api/v1/databases/create", "Bearer "+owner.Token, request,
			)
			require.Equal(t, http.StatusCreated, createDatabaseResponse.Code,
				"create failed: %s", createDatabaseResponse.Body.String())

			var database Database
			require.NoError(t, json.Unmarshal(createDatabaseResponse.Body.Bytes(), &database))
			defer func() {
				RemoveTestDatabase(t.Context(), &database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			_, err := conn.Exec(t.Context(),
				fmt.Sprintf(`REVOKE EXECUTE ON FUNCTION pg_switch_wal() FROM "%s"`, username))
			require.NoError(t, err)

			connectionTestResponse := test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/"+database.ID.String()+"/test-connection",
				"Bearer "+owner.Token,
				nil,
				http.StatusBadRequest,
			)

			assert.Equal(t,
				postgresql_shared.ConnErrNoWalSwitchPrivilege,
				getConnectionErrorCode(t, connectionTestResponse.Body))
		})
	}
}

func Test_UpdateDatabase_RaisingToWalStreamWhenRoleCannotSwitchWal_ReturnsNoWalSwitchPrivilegeCode(
	t *testing.T,
) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			username, password := createReplicationRole(t, openFixtureConn(t, fixture), false)

			request := buildPhysicalDatabaseRequestFor(t, physicalDatabaseRequestSpec{
				workspaceID: workspace.ID,
				fixture:     fixture,
				username:    username,
				password:    password,
				backupType:  postgresql_physical.BackupTypeFullAndIncremental,
			})

			createDatabaseResponse := workspaces_testing.MakeAPIRequest(
				router, "POST", "/api/v1/databases/create", "Bearer "+owner.Token, request,
			)
			require.Equal(t, http.StatusCreated, createDatabaseResponse.Code,
				"create failed: %s", createDatabaseResponse.Body.String())

			var database Database
			require.NoError(t, json.Unmarshal(createDatabaseResponse.Body.Bytes(), &database))
			defer func() {
				RemoveTestDatabase(t.Context(), &database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			database.PostgresqlPhysical.Password = password
			database.PostgresqlPhysical.BackupType = postgresql_physical.BackupTypeFullIncrementalAndWalStream

			updateDatabaseResponse := workspaces_testing.MakeAPIRequest(
				router, "POST", "/api/v1/databases/update", "Bearer "+owner.Token, database,
			)

			require.Equal(t, http.StatusBadRequest, updateDatabaseResponse.Code,
				"expected refusal, got: %s", updateDatabaseResponse.Body.String())

			var updateDatabaseErrorResponse struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal(updateDatabaseResponse.Body.Bytes(), &updateDatabaseErrorResponse))

			// Create and update surface only Error(), so this path shows the remedy sentence
			// rather than the machine-readable code the test-connection endpoints return.
			assert.Contains(t, updateDatabaseErrorResponse.Error,
				"GRANT EXECUTE ON FUNCTION pg_switch_wal()")
			assert.Contains(t, updateDatabaseErrorResponse.Error,
				pgx.Identifier{username}.Sanitize())
		})
	}
}
