package backups_controllers_logical

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/download/stream_guard"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	users_middleware "databasus-backend/internal/features/users/middleware"
	files_utils "databasus-backend/internal/util/files"
)

type BackupController struct {
	backupService *backups_services.LogicalBackupService
}

func (c *BackupController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/backups", c.GetBackups)
	router.POST("/backups", c.MakeBackup)
	router.POST("/backups/:id/download-token", c.GenerateDownloadToken)
	router.DELETE("/backups/:id", c.DeleteBackup)
	router.POST("/backups/:id/cancel", c.CancelBackup)
}

// RegisterPublicRoutes registers routes that don't require Bearer authentication
// (they have their own authentication mechanisms like download tokens)
func (c *BackupController) RegisterPublicRoutes(router *gin.RouterGroup) {
	router.GET("/backups/:id/file", c.GetFile)
}

func (c *BackupController) GetBackups(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request backups_dto_logical.GetBackupsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	databaseID, err := uuid.Parse(request.DatabaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database_id"})
		return
	}

	filters := c.buildBackupFilters(&request)

	response, err := c.backupService.GetBackups(
		ctx.Request.Context(),
		user,
		databaseID,
		request.Limit,
		request.Offset,
		filters,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *BackupController) MakeBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request backups_dto_logical.MakeBackupRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.backupService.MakeBackupWithAuth(ctx.Request.Context(), user, request.DatabaseID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "backup started successfully"})
}

func (c *BackupController) DeleteBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.backupService.DeleteBackup(ctx.Request.Context(), user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *BackupController) CancelBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.backupService.CancelBackup(ctx.Request.Context(), user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *BackupController) GenerateDownloadToken(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	response, err := c.backupService.GenerateDownloadToken(ctx.Request.Context(), user, id)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(
				http.StatusConflict,
				gin.H{
					"error": "Download already in progress for some of backups. Please wait until previous download completed or cancel it",
				},
			)
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *BackupController) GetFile(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "download token is required"})
		return
	}

	backupIDParam := ctx.Param("id")
	backupID, err := uuid.Parse(backupIDParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	downloadToken, err := c.backupService.ValidateDownloadToken(ctx.Request.Context(), token)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(
				http.StatusConflict,
				gin.H{
					"error": "download already in progress for this user. Please wait until previous download completed or cancel it",
				},
			)
			return
		}

		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired download token"})
		return
	}

	if downloadToken.BackupID != backupID {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired download token"})
		return
	}

	fileReader, backup, database, err := c.backupService.GetBackupFileWithoutAuth(
		ctx.Request.Context(),
		downloadToken.BackupID,
	)
	if err != nil {
		c.backupService.ReleaseDownloadLock(ctx.Request.Context(), downloadToken.UserID)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	defer func() {
		cancelHeartbeat()
		c.backupService.ReleaseDownloadLock(ctx.Request.Context(), downloadToken.UserID)
		if err := fileReader.Close(); err != nil {
			fmt.Printf("Error closing file reader: %v\n", err)
		}
	}()

	go c.startDownloadHeartbeat(heartbeatCtx, downloadToken.UserID)

	filename := c.generateBackupFilename(backup, database)

	if backup.BackupSizeMb > 0 {
		sizeBytes := int64(backup.BackupSizeMb * 1024 * 1024)
		ctx.Header("Content-Length", fmt.Sprintf("%d", sizeBytes))
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename),
	)

	_, err = io.Copy(ctx.Writer, fileReader)
	if err != nil {
		fmt.Printf("Error streaming file: %v\n", err)
	}

	c.backupService.WriteAuditLogForDownload(ctx.Request.Context(), downloadToken.UserID, backup, database)
}

func (c *BackupController) generateBackupFilename(
	backup *backups_core_logical.LogicalBackup,
	database *databases.Database,
) string {
	// Format timestamp as YYYY-MM-DD_HH-mm-ss
	timestamp := backup.CreatedAt.Format("2006-01-02_15-04-05")

	// Sanitize database name for filename (replace spaces and special chars)
	safeName := files_utils.SanitizeFilename(database.Name)

	// Determine extension based on database type
	extension := c.getBackupExtension(database.Type)

	return fmt.Sprintf("%s_backup_%s%s", safeName, timestamp, extension)
}

func (c *BackupController) getBackupExtension(
	dbType databases.DatabaseType,
) string {
	switch dbType {
	case databases.DatabaseTypeMysql, databases.DatabaseTypeMariadb:
		return ".sql.zst"
	case databases.DatabaseTypePostgresLogical:
		// PostgreSQL custom format
		return ".dump"
	case databases.DatabaseTypeMongodb:
		return ".archive"
	default:
		return ".backup"
	}
}

func (c *BackupController) startDownloadHeartbeat(ctx context.Context, userID uuid.UUID) {
	ticker := time.NewTicker(stream_guard.GetDownloadHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.backupService.RefreshDownloadLock(ctx, userID)
		}
	}
}

func (c *BackupController) buildBackupFilters(
	request *backups_dto_logical.GetBackupsRequest,
) *backups_core_logical.BackupFilters {
	isHasFilters := len(request.Statuses) > 0 ||
		request.BeforeDate != nil

	if !isHasFilters {
		return nil
	}

	filters := &backups_core_logical.BackupFilters{}

	if len(request.Statuses) > 0 {
		statuses := make([]backups_core_logical.BackupStatus, 0, len(request.Statuses))
		for _, statusStr := range request.Statuses {
			statuses = append(statuses, backups_core_logical.BackupStatus(statusStr))
		}

		filters.Statuses = statuses
	}

	filters.BeforeDate = request.BeforeDate

	return filters
}
