package backups_config_physical

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
)

type BackupConfigController struct {
	backupConfigService *BackupConfigService
}

func (c *BackupConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/backup-configs/physical/save", c.SaveBackupConfig)
	router.GET("/backup-configs/physical/database/:id", c.GetBackupConfigByDbID)
	router.POST("/backup-configs/physical/database/:id/transfer", c.TransferDatabase)
}

func (c *BackupConfigController) SaveBackupConfig(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var requestDTO PhysicalBackupConfig
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requestDTO.StorageID = nil

	savedConfig, err := c.backupConfigService.SaveBackupConfigWithAuth(ctx.Request.Context(), actor, &requestDTO)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, savedConfig)
}

func (c *BackupConfigController) GetBackupConfigByDbID(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	backupConfig, err := c.backupConfigService.GetAndRepairBackupConfigByDbIdWithAuth(ctx.Request.Context(), actor, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "backup configuration not found"})
		return
	}

	ctx.JSON(http.StatusOK, backupConfig)
}

func (c *BackupConfigController) TransferDatabase(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request TransferDatabaseRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.TargetWorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "targetWorkspaceId is required"})
		return
	}

	if err := c.backupConfigService.TransferDatabaseToWorkspace(
		ctx.Request.Context(),
		actor,
		id,
		&request,
	); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsInSourceWorkspace) ||
			errors.Is(err, ErrInsufficientPermissionsInTargetWorkspace) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "database transferred successfully"})
}
