package backups_config_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func createCrossStrategyRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config_logical.GetBackupConfigController(),
		backups_config_physical.GetBackupConfigController(),
		storages.GetStorageController(),
		notifiers.GetNotifierController(),
	)

	storages.SetupDependencies()
	databases.SetupDependencies()
	notifiers.SetupDependencies()
	backups_config_logical.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()

	return router
}

func Test_StorageDeletion_WithPhysicalOnlyAttachment_BlocksDeletion(t *testing.T) {
	router := createCrossStrategyRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	physicalDatabase := databases.CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	storage := storages.CreateTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), physicalDatabase)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	backups_config_physical.EnableBackupsForPhysicalTestDatabase(t.Context(), physicalDatabase.ID, storage)

	testResp := test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/storages/%s", storage.ID.String()),
		"Bearer "+owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "storage has attached databases")
}

func Test_StorageDeletion_WithLogicalOnlyAttachment_BlocksDeletion(t *testing.T) {
	router := createCrossStrategyRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	storage := storages.CreateTestStorage(workspace.ID)
	logicalDatabase := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), logicalDatabase)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), logicalDatabase.ID, storage)

	testResp := test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/storages/%s", storage.ID.String()),
		"Bearer "+owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "storage has attached databases")
}

func Test_StorageIsUsing_WithPhysicalOnlyAttachment_ReportsTrue(t *testing.T) {
	router := createCrossStrategyRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	physicalDatabase := databases.CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	storage := storages.CreateTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), physicalDatabase)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	backups_config_physical.EnableBackupsForPhysicalTestDatabase(t.Context(), physicalDatabase.ID, storage)

	var response map[string]bool
	test_utils.MakeGetRequestAndUnmarshal(
		t, router,
		fmt.Sprintf("/api/v1/backup-configs/storage/%s/is-using", storage.ID),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.True(t, response["isUsing"], "expected is-using=true when physical config attached")
}

func Test_StorageDeletion_WithNoAttachment_Succeeds(t *testing.T) {
	router := createCrossStrategyRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	storage := storages.CreateTestStorage(workspace.ID)

	defer func() {
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/storages/%s", storage.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
	)
}
