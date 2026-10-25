package notifiers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
)

type NotifierController struct {
	notifierService  *NotifierService
	workspaceService *workspaces_services.WorkspaceService
}

func (c *NotifierController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/notifiers", c.SaveNotifier)
	router.GET("/notifiers", c.GetNotifiers)
	router.GET("/notifiers/:id", c.GetNotifier)
	router.DELETE("/notifiers/:id", c.DeleteNotifier)
	router.POST("/notifiers/:id/test", c.SendTestNotification)
	router.POST("/notifiers/:id/transfer", c.TransferNotifierToWorkspace)
	router.POST("/notifiers/direct-test", c.SendTestNotificationDirect)
}

func (c *NotifierController) SaveNotifier(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Notifier
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	if err := c.notifierService.SaveNotifier(ctx.Request.Context(), actor, request.WorkspaceID, &request); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToManageNotifier) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, request)
}

func (c *NotifierController) GetNotifier(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	notifier, err := c.notifierService.GetNotifier(ctx.Request.Context(), actor, id)
	if err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToViewNotifier) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, notifier)
}

func (c *NotifierController) GetNotifiers(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
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

	notifiers, err := c.notifierService.GetNotifiers(ctx.Request.Context(), actor, workspaceID)
	if err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToViewNotifiers) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, notifiers)
}

func (c *NotifierController) DeleteNotifier(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	if err := c.notifierService.DeleteNotifier(ctx.Request.Context(), actor, id); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToManageNotifier) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "notifier deleted successfully"})
}

func (c *NotifierController) SendTestNotification(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	if err := c.notifierService.SendTestNotification(ctx.Request.Context(), actor, id); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToTestNotifier) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "test notification sent successfully"})
}

func (c *NotifierController) TransferNotifierToWorkspace(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	var request TransferNotifierRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.TargetWorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "targetWorkspaceId is required"})
		return
	}

	if err := c.notifierService.TransferNotifierToWorkspace(
		ctx.Request.Context(),
		actor,
		id,
		request.TargetWorkspaceID,
		nil,
	); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsInSourceWorkspace) ||
			errors.Is(err, ErrInsufficientPermissionsInTargetWorkspace) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "notifier transferred successfully"})
}

func (c *NotifierController) SendTestNotificationDirect(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Notifier
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	canView, _, err := c.workspaceService.CanUserAccessWorkspace(ctx.Request.Context(), request.WorkspaceID, actor)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !canView {
		ctx.JSON(
			http.StatusForbidden,
			gin.H{"error": "insufficient permissions to test notifier in this workspace"},
		)
		return
	}

	if err := c.notifierService.SendTestNotificationToNotifier(ctx.Request.Context(), &request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "test notification sent successfully"})
}
