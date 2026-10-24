//go:build testhelpers

package logicaltesting

import (
	"github.com/gin-gonic/gin"

	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/restores"
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
		restores.GetRestoreController(),
	)

	backups_controllers_logical.GetBackupController().RegisterPublicRoutes(router.Group("/api/v1"))

	return router
}
