package audit_logs

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	user_models "databasus-backend/internal/features/users/models"
)

type AuditLogController struct {
	auditLogService *AuditLogService
}

func (c *AuditLogController) RegisterRoutes(router *gin.RouterGroup) {
	// All audit log endpoints require authentication (handled in main.go)
	auditRoutes := router.Group("/audit-logs")

	auditRoutes.GET("/global", c.GetGlobalAuditLogs)
	auditRoutes.GET("/users/:userId", c.GetUserAuditLogs)
}

func (c *AuditLogController) GetGlobalAuditLogs(ctx *gin.Context) {
	user, isOk := ctx.MustGet("user").(*user_models.User)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	request := &GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.auditLogService.GetGlobalAuditLogs(ctx.Request.Context(), user, request)
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
	user, isOk := ctx.MustGet("user").(*user_models.User)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
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

	response, err := c.auditLogService.GetUserAuditLogs(ctx.Request.Context(), targetUserID, user, request)
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
