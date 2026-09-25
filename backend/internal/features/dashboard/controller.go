package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
)

type DashboardController struct {
	dashboardService *DashboardService
}

func (c *DashboardController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/dashboard", c.GetWorkspaceDashboard)

	adminOnly := router.Group("/dashboard")
	adminOnly.Use(users_middleware.RequireRole(users_enums.UserRoleAdmin))

	adminOnly.GET("/installation", c.GetInstallationDashboard)
}

// GetWorkspaceDashboard
// @Summary Get workspace dashboard
// @Description Lists every database of a workspace with its health, its last healthcheck attempts and the totals of the backups it currently stores, plus the workspace totals. Sizes count completed backups only; for physical databases they also include WAL segments. The backup count includes every stored backup regardless of status, and for physical databases counts FULL and INCREMENTAL backups but not WAL segments. Any workspace member can read it.
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Param workspace_id query string true "Workspace ID"
// @Success 200 {object} WorkspaceDashboard
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /dashboard [get]
func (c *DashboardController) GetWorkspaceDashboard(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Query("workspace_id")
	if workspaceIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id query parameter is required"})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	dashboard, err := c.dashboardService.GetWorkspaceDashboard(ctx.Request.Context(), user, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, dashboard)
}

// GetInstallationDashboard
// @Summary Get installation dashboard (ADMIN only)
// @Description Returns the database count and backup totals across every workspace, including databases created by restores that belong to no workspace. Sizes and counts follow the same rules as the workspace dashboard.
// @Tags dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DashboardTotals
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dashboard/installation [get]
func (c *DashboardController) GetInstallationDashboard(ctx *gin.Context) {
	totals, err := c.dashboardService.GetInstallationDashboard()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, totals)
}
