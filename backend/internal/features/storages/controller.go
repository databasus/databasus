package storages

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_middleware "databasus-backend/internal/features/users/middleware"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

type StorageController struct {
	storageService   *StorageService
	workspaceService *workspaces_services.WorkspaceService
}

func (c *StorageController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/storages", c.SaveStorage)
	router.GET("/storages", c.GetStorages)
	router.GET("/storages/:id", c.GetStorage)
	router.DELETE("/storages/:id", c.DeleteStorage)
	router.POST("/storages/:id/test", c.TestStorageConnection)
	router.POST("/storages/:id/transfer", c.TransferStorageToWorkspace)
	router.POST("/storages/direct-test", c.TestStorageConnectionDirect)
}

func (c *StorageController) SaveStorage(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Storage
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	if err := c.storageService.SaveStorage(ctx.Request.Context(), user, request.WorkspaceID, &request); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToManageStorage) ||
			errors.Is(err, ErrRcloneStorageRequiresAdmin) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, request)
}

func (c *StorageController) GetStorage(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage ID"})
		return
	}

	storage, err := c.storageService.GetStorage(ctx.Request.Context(), user, id)
	if err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToViewStorage) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, storage)
}

func (c *StorageController) GetStorages(ctx *gin.Context) {
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

	storages, err := c.storageService.GetStorages(ctx.Request.Context(), user, workspaceID)
	if err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToViewStorages) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, storages)
}

func (c *StorageController) DeleteStorage(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage ID"})
		return
	}

	if err := c.storageService.DeleteStorage(ctx.Request.Context(), user, id); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToManageStorage) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "storage deleted successfully"})
}

func (c *StorageController) TestStorageConnection(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage ID"})
		return
	}

	if err := c.storageService.TestStorageConnection(ctx.Request.Context(), user, id); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsToTestStorage) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "storage connection test successful"})
}

func (c *StorageController) TransferStorageToWorkspace(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage ID"})
		return
	}

	var request TransferStorageRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.TargetWorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "targetWorkspaceId is required"})
		return
	}

	if err := c.storageService.TransferStorageToWorkspace(
		ctx.Request.Context(),
		user,
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

	ctx.JSON(http.StatusOK, gin.H{"message": "storage transferred successfully"})
}

func (c *StorageController) TestStorageConnectionDirect(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Storage
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	canView, _, err := c.workspaceService.CanUserAccessWorkspace(ctx.Request.Context(), request.WorkspaceID, user)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !canView {
		ctx.JSON(
			http.StatusForbidden,
			gin.H{"error": "insufficient permissions to test storage in this workspace"},
		)
		return
	}

	if err := c.storageService.TestStorageConnectionDirect(ctx.Request.Context(), user, &request); err != nil {
		if errors.Is(err, ErrRcloneStorageRequiresAdmin) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "storage connection test successful"})
}
