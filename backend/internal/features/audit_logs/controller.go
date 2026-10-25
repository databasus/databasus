package audit_logs

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type AuditLogController struct {
	auditLogService *AuditLogService
	workspaceAccess *workspaces_bridge.Access
}

func (c *AuditLogController) RegisterRoutes(router *gin.RouterGroup) {
	// All audit log endpoints require authentication (handled in main.go)
	auditRoutes := router.Group("/audit-logs")

	auditRoutes.GET("/global", c.GetGlobalAuditLogs)
	auditRoutes.GET("/users/:userId", c.GetUserAuditLogs)

	// The parameter shares its name with the workspace routes of the contract, because Gin refuses
	// two wildcard names in one path segment.
	router.GET("/workspaces/:workspaceId/audit-logs", c.GetWorkspaceAuditLogs)
}

func (c *AuditLogController) GetGlobalAuditLogs(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	request := &GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.auditLogService.GetGlobalAuditLogs(ctx.Request.Context(), actor, request)
	if err != nil {
		if errors.Is(err, ErrOnlyAdminsCanViewGlobalLogs) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		_ = ctx.Error(err)

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *AuditLogController) GetUserAuditLogs(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := ctx.Param("userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	request := &GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.auditLogService.GetUserAuditLogs(ctx.Request.Context(), targetUserID, actor, request)
	if err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToViewLogs) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		_ = ctx.Error(err)

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *AuditLogController) GetWorkspaceAuditLogs(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceID, err := uuid.Parse(ctx.Param("workspaceId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	request := &GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	isAllowed, err := c.workspaceAccess.IsActorAllowed(ctx.Request.Context(), actor, workspaceID, workspaces.ActionView)
	if err != nil {
		_ = ctx.Error(err)

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	if !isAllowed {
		ctx.JSON(http.StatusForbidden, gin.H{"error": ErrInsufficientPermissionsToViewWorkspaceLogs.Error()})
		return
	}

	response, err := c.auditLogService.GetWorkspaceAuditLogs(ctx.Request.Context(), workspaceID, request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
