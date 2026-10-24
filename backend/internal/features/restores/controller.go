package restores

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	restores_core "databasus-backend/internal/features/restores/core"
	users_middleware "databasus-backend/internal/features/users/middleware"
)

type RestoreController struct {
	restoreService *RestoreService
}

func (c *RestoreController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/restores/:backupId", c.GetRestores)
	router.POST("/restores/:backupId/restore", c.RestoreBackup)
	router.POST("/restores/cancel/:restoreId", c.CancelRestore)
}

func (c *RestoreController) GetRestores(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	restores, err := c.restoreService.GetRestores(ctx.Request.Context(), user, backupID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, restores)
}

func (c *RestoreController) RestoreBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	var requestDTO restores_core.RestoreBackupRequest
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.restoreService.RestoreBackupWithAuth(ctx.Request.Context(), user, backupID, requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "restore started successfully"})
}

func (c *RestoreController) CancelRestore(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	restoreID, err := uuid.Parse(ctx.Param("restoreId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid restore ID"})
		return
	}

	if err := c.restoreService.CancelRestore(ctx.Request.Context(), user, restoreID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}
