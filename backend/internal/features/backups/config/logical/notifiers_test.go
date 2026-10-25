package backups_config_logical

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_AttachNotifierFromSameWorkspace_SuccessfullyAttached(t *testing.T) {
	router := createTestRouterWithNotifier()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	database.Notifiers = []notifiers.Notifier{*notifier}

	var response databases.Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.ID)
	assert.Len(t, response.Notifiers, 1)
	assert.Equal(t, notifier.ID, response.Notifiers[0].ID)
}

func Test_AttachNotifierFromDifferentWorkspace_ReturnsForbidden(t *testing.T) {
	router := createTestRouterWithNotifier()

	owner1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", owner1, router)
	database := createTestDatabaseViaAPI("Test Database", workspace1.ID, owner1.Token, router)

	owner2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", owner2, router)
	notifier := notifiers.CreateTestNotifier(workspace2.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2, router)
	}()

	database.Notifiers = []notifiers.Notifier{*notifier}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner1.Token,
		database,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "notifier does not belong to this workspace")
}

func Test_DeleteNotifierWithAttachedDatabases_CannotDelete(t *testing.T) {
	router := createTestRouterWithNotifier()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	database.Notifiers = []notifiers.Notifier{*notifier}

	var response databases.Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database,
		http.StatusOK,
		&response,
	)

	testResp := test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", notifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(
		t,
		string(testResp.Body),
		"notifier has attached databases and cannot be deleted",
	)
}

func Test_TransferNotifierWithAttachedDatabase_CannotTransfer(t *testing.T) {
	router := createTestRouterWithNotifier()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Target Workspace", owner, router)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
	}()

	database.Notifiers = []notifiers.Notifier{*notifier}

	var response databases.Database
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/databases/update",
		"Bearer "+owner.Token,
		database,
		http.StatusOK,
		&response,
	)

	transferRequest := notifiers.TransferNotifierRequest{
		TargetWorkspaceID: targetWorkspace.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s/transfer", notifier.ID.String()),
		"Bearer "+owner.Token,
		transferRequest,
		http.StatusBadRequest,
	)

	assert.Contains(
		t,
		string(testResp.Body),
		"notifier has attached databases and cannot be transferred",
	)
}

func createTestRouterWithNotifier() *gin.Engine {
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
