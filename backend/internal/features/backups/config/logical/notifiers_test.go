package backups_config_logical

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/users"
)

func Test_AttachNotifierFromSameWorkspace_SuccessfullyAttached(t *testing.T) {
	router := createTestRouterWithNotifier()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := databases_bridge.CreateTestDatabaseViaAPI(t, router, owner.Token, workspace.ID, "Test Database")
	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)

	defer func() {
		databases_bridge.RemoveTestDatabase(t.Context(), database)
		notifiers_bridge.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	attachNotifiersViaAPI(t, router, owner.Token, database.ID, http.StatusOK, notifier.ID)

	updatedDatabase := databases_bridge.GetTestDatabaseViaAPI(t, router, owner.Token, database.ID)
	assert.Equal(t, []uuid.UUID{notifier.ID}, updatedDatabase.NotifierIds)
}

func Test_AttachNotifierFromDifferentWorkspace_ReturnsForbidden(t *testing.T) {
	router := createTestRouterWithNotifier()

	owner1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", owner1)
	database := databases_bridge.CreateTestDatabaseViaAPI(t, router, owner1.Token, workspace1.ID, "Test Database")

	owner2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", owner2)
	notifier := notifiers_bridge.CreateTestNotifier(workspace2.ID)

	defer func() {
		databases_bridge.RemoveTestDatabase(t.Context(), database)
		notifiers_bridge.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2)
	}()

	response := databases_bridge.UpdateTestDatabaseViaAPI(t, router, owner1.Token, database.ID,
		func(update *apiclient.DatabaseUpdate) { update.NotifierIds = []uuid.UUID{notifier.ID} })

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "must name notifiers of the database's workspace")
}

func createTestRouterWithNotifier() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		databases_bridge.GetTestRoutes(),
		GetBackupConfigController(),
	)

	databases_bridge.SetupDependencies()
	SetupDependencies()

	return router
}

func attachNotifiersViaAPI(
	t *testing.T,
	router *gin.Engine,
	token string,
	databaseID uuid.UUID,
	expectedStatus int,
	notifierIDs ...uuid.UUID,
) {
	t.Helper()

	response := databases_bridge.UpdateTestDatabaseViaAPI(t, router, token, databaseID,
		func(update *apiclient.DatabaseUpdate) { update.NotifierIds = notifierIDs })
	require.Equalf(t, expectedStatus, response.Code, "update database notifiers: %s", response.Body.String())
}
