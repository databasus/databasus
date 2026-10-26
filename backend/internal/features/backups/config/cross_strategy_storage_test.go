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
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func createCrossStrategyRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		databases.GetDatabaseController(),
		backups_config_logical.GetBackupConfigController(),
		backups_config_physical.GetBackupConfigController(),
	)

	databases.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()

	return router
}

func Test_StorageIsUsing_WithPhysicalOnlyAttachment_ReportsTrue(t *testing.T) {
	router := createCrossStrategyRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)
	notifier := notifiers_bridge.CreateTestNotifier(workspace.ID)

	physicalDatabase := databases.CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")
	storage := storages_bridge.CreateTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), physicalDatabase)
		storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
		notifiers_bridge.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	backups_config_physical.EnableBackupsForPhysicalTestDatabase(t.Context(), physicalDatabase.ID, storage.ID)

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
