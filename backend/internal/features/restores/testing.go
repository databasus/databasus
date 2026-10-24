//go:build testhelpers

package restores

import (
	"github.com/gin-gonic/gin"

	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

func CreateTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config_logical.GetBackupConfigController(),
		backups_controllers_logical.GetBackupController(),
		GetRestoreController(),
	)

	v1 := router.Group("/api/v1")
	backups_controllers_logical.GetBackupController().RegisterPublicRoutes(v1)

	return router
}
