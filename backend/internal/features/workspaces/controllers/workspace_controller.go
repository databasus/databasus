package workspaces_controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_errors "databasus-backend/internal/features/workspaces/errors"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
)

type WorkspaceController struct {
	workspaceService *workspaces_services.WorkspaceService
}

func (c *WorkspaceController) RegisterRoutes(router *gin.RouterGroup) {
	workspaceRoutes := router.Group("/workspaces")

	workspaceRoutes.POST("", c.CreateWorkspace)
	workspaceRoutes.GET("", c.GetWorkspaces)
	workspaceRoutes.GET("/:id", c.GetWorkspace)
	workspaceRoutes.PUT("/:id", c.UpdateWorkspace)
	workspaceRoutes.DELETE("/:id", c.DeleteWorkspace)
	workspaceRoutes.GET("/:id/audit-logs", c.GetWorkspaceAuditLogs)
}

func (c *WorkspaceController) CreateWorkspace(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request workspaces_dto.CreateWorkspaceRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.workspaceService.CreateWorkspace(ctx.Request.Context(), &request, actor)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToCreateWorkspaces) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *WorkspaceController) GetWorkspaces(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	response, err := c.workspaceService.GetUserWorkspaces(ctx.Request.Context(), actor)
	if err != nil {
		_ = ctx.Error(err)

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workspaces"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *WorkspaceController) GetWorkspace(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	workspace, err := c.workspaceService.GetWorkspace(ctx.Request.Context(), workspaceID, actor)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToViewWorkspace) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, workspace)
}

func (c *WorkspaceController) UpdateWorkspace(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var workspace workspaces_models.Workspace
	if err := ctx.ShouldBindJSON(&workspace); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	updatedWorkspace, err := c.workspaceService.UpdateWorkspace(ctx.Request.Context(), workspaceID, &workspace, actor)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToUpdateWorkspace) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedWorkspace)
}

func (c *WorkspaceController) DeleteWorkspace(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	if err := c.workspaceService.DeleteWorkspace(ctx.Request.Context(), workspaceID, actor); err != nil {
		if errors.Is(err, workspaces_errors.ErrOnlyOwnerOrAdminCanDeleteWorkspace) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Workspace deleted successfully"})
}

func (c *WorkspaceController) GetWorkspaceAuditLogs(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	request := &audit_logs.GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.workspaceService.GetWorkspaceAuditLogs(ctx.Request.Context(), workspaceID, actor, request)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToViewWorkspaceAuditLogs) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
