//go:build testhelpers

package logicaltesting

import (
	"github.com/gin-gonic/gin"

	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/restores"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

func CreateTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		databases_bridge.GetTestRoutes(),
		backups_bridge.GetTestRoutes(),
		backups_controllers_logical.GetBackupController(),
		restores.GetRestoreController(),
	)

	backups_controllers_logical.GetBackupController().RegisterPublicRoutes(router.Group("/api/v1"))

	return router
}
