package backups_config_logical

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_AttachNotifierFromSameWorkspace_SuccessfullyAttached(t *testing.T) {
	router := createTestRouterWithNotifier()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers_bridge.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	database.Notifiers = []databases.DatabaseNotifier{{ID: notifier.ID}}

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
	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", owner1)
	database := createTestDatabaseViaAPI("Test Database", workspace1.ID, owner1.Token, router)

	owner2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", owner2)
	notifier := notifiers_bridge.CreateTestNotifier(workspace2.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers_bridge.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2)
	}()

	database.Notifiers = []databases.DatabaseNotifier{{ID: notifier.ID}}

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

func createTestRouterWithNotifier() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		databases.GetDatabaseController(),
		GetBackupConfigController(),
	)

	databases.SetupDependencies()
	SetupDependencies()

	return router
}
