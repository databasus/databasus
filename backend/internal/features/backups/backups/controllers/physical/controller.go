package backups_controllers_physical

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	"databasus-backend/internal/features/backups/backups/download/restore_token"
	"databasus-backend/internal/features/backups/backups/download/stream_guard"
	backups_dto_physical "databasus-backend/internal/features/backups/backups/dto/physical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_models "databasus-backend/internal/features/users/models"
)

type PhysicalBackupController struct {
	physicalBackupService *backups_services.PhysicalBackupService
	restoreTokenService   *restore_token.Service
}

func (c *PhysicalBackupController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/backups/physical/database/:id/backups", c.GetBackups)
	router.GET("/backups/physical/database/:id/wal-stream/statistics", c.GetWalStreamStatistics)
	router.GET("/backups/physical/database/:id/wal-stream/live", c.GetWalStreamLive)
	router.POST("/backups/physical/database/:id/trigger", c.TriggerBackup)
	router.POST("/backups/physical/database/:id/restore-token", c.GenerateRestoreToken)

	router.POST("/backups/physical/backups/:backupId/restore-token", c.GenerateBackupRestoreToken)
	router.POST("/backups/physical/backups/:backupId/cancel", c.CancelBackup)
	router.DELETE("/backups/physical/backups/:backupId", c.DeleteBackup)
}

func (c *PhysicalBackupController) RegisterPublicRoutes(router *gin.RouterGroup) {
	router.GET("/backups/physical/restore-stream", c.GetRestoreStream)
	router.GET("/backups/physical/recovery-script", c.GetRecoveryScript)
}

func (c *PhysicalBackupController) GetRecoveryScript(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/x-shellscript; charset=utf-8")
	ctx.Header("Content-Disposition", `inline; filename="databasus-recovery.sh"`)
	ctx.String(http.StatusOK, recoveryScript)
}

func (c *PhysicalBackupController) GetBackups(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups_dto_physical.GetPhysicalBackupsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.physicalBackupService.GetBackups(ctx.Request.Context(), user, databaseID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *PhysicalBackupController) GetWalStreamLive(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	response, err := c.physicalBackupService.GetWalStreamLive(ctx.Request.Context(), user, databaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *PhysicalBackupController) GetWalStreamStatistics(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups_dto_physical.GetWalStreamStatisticsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.physicalBackupService.GetWalStreamStatistics(ctx.Request.Context(), user, databaseID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *PhysicalBackupController) TriggerBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups_dto_physical.TriggerBackupRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.requestBackupOfType(ctx.Request.Context(), user, databaseID, request.Type); err != nil {
		if errors.Is(err, backups_services.ErrNoExtendableChain) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "backup requested"})
}

func (c *PhysicalBackupController) GenerateRestoreToken(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups_dto_physical.GenerateRestoreTokenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := c.physicalBackupService.GenerateRestoreToken(ctx.Request.Context(), user, databaseID, &request)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		// An unreachable target (WAL gap, no chain, target before earliest) is
		// resolved at issue time, so surface it here with the same status the
		// stream endpoint would have returned.
		if status, message, isResolverError := classifyRestoreStreamError(err); isResolverError {
			ctx.JSON(status, gin.H{"error": message})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, backups_dto_physical.GenerateRestoreTokenResponse{
		Token: token,
		URL:   "/api/v1/backups/physical/restore-stream?token=" + token,
	})
}

func (c *PhysicalBackupController) GenerateBackupRestoreToken(ctx *gin.Context) {
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

	token, err := c.physicalBackupService.GenerateBackupRestoreToken(ctx.Request.Context(), user, backupID)
	if err != nil {
		if errors.Is(err, backups_services.ErrBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		// A non-restorable backup (missing chain, not COMPLETED) is resolved up
		// front; surface it with the same status the stream endpoint would.
		if status, message, isResolverError := classifyRestoreStreamError(err); isResolverError {
			ctx.JSON(status, gin.H{"error": message})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, backups_dto_physical.GenerateRestoreTokenResponse{
		Token: token,
		URL:   "/api/v1/backups/physical/restore-stream?token=" + token,
	})
}

func (c *PhysicalBackupController) CancelBackup(ctx *gin.Context) {
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

	if err := c.physicalBackupService.CancelBackup(ctx.Request.Context(), user, backupID); err != nil {
		if errors.Is(err, backups_services.ErrBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *PhysicalBackupController) DeleteBackup(ctx *gin.Context) {
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

	if err := c.physicalBackupService.DeleteBackup(ctx.Request.Context(), user, backupID); err != nil {
		if errors.Is(err, backups_services.ErrBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *PhysicalBackupController) GetRestoreStream(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "restore token is required"})
		return
	}

	restoreToken, err := c.restoreTokenService.ValidateAndConsumeRestoreToken(ctx.Request.Context(), token)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "a download or restore is already in progress for this user",
			})
			return
		}

		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired restore token"})
		return
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	defer func() {
		cancelHeartbeat()
		c.restoreTokenService.ReleaseDownloadLock(ctx.Request.Context(), restoreToken.UserID)
	}()

	go c.startStreamHeartbeat(heartbeatCtx, restoreToken.UserID)

	ctx.Header("Content-Type", "application/x-tar")
	ctx.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename="restore-%s.tar"`, restoreToken.DatabaseID))

	if err := c.openRestoreStream(ctx.Request.Context(), restoreToken, ctx.Writer); err != nil {
		// Resolution runs before any bytes are written, so if nothing has been
		// flushed yet we can still answer with a proper status. Once the tar has
		// started, the status is already 200 and we can only log.
		if !ctx.Writer.Written() {
			status, message := mapRestoreStreamError(err)
			ctx.JSON(status, gin.H{"error": message})
		}

		return
	}
}

// mapRestoreStreamError translates resolver failures into HTTP statuses for the
// stream endpoint, where an unrecognized error is an internal failure (500).
func mapRestoreStreamError(err error) (int, string) {
	if status, message, isResolverError := classifyRestoreStreamError(err); isResolverError {
		return status, message
	}

	return http.StatusInternalServerError, err.Error()
}

// classifyRestoreStreamError maps the resolver's typed failures to HTTP statuses
// and reports whether err was one of them. A WAL gap or out-of-range target is
// the caller's mistake to correct (422); no chain is a not-found (404). The bool
// lets callers fall through to their own default for everything else.
func classifyRestoreStreamError(err error) (int, string, bool) {
	var gapErr chain_view.WalGapBeforeTargetError
	if errors.As(err, &gapErr) {
		return http.StatusUnprocessableEntity, gapErr.Error(), true
	}

	if errors.Is(err, chain_view.ErrTargetBeforeEarliest) {
		return http.StatusUnprocessableEntity, err.Error(), true
	}

	if errors.Is(err, chain_view.ErrNoChainForRestore) {
		return http.StatusNotFound, err.Error(), true
	}

	return 0, "", false
}

// openRestoreStream dispatches by what the token carries: a BackupID streams a
// per-backup restore (FULL + incremental ancestors, no WAL); otherwise
// TargetTime drives a point-in-time restore.
func (c *PhysicalBackupController) openRestoreStream(
	streamCtx context.Context,
	token *restore_token.Token,
	w io.Writer,
) error {
	if token.BackupID != nil {
		return c.physicalBackupService.OpenRestoreStreamForBackup(streamCtx, token.DatabaseID, *token.BackupID, w)
	}

	return c.physicalBackupService.OpenRestoreStream(streamCtx, token.DatabaseID, token.TargetTime, w)
}

func (c *PhysicalBackupController) requestBackupOfType(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
	backupType backups_dto_physical.TriggerBackupType,
) error {
	switch backupType {
	case backups_dto_physical.TriggerBackupTypeFull:
		return c.physicalBackupService.RequestFullBackup(ctx, user, databaseID)

	case backups_dto_physical.TriggerBackupTypeIncremental:
		return c.physicalBackupService.RequestIncrementalBackup(ctx, user, databaseID)

	default:
		return c.physicalBackupService.RequestBackup(ctx, user, databaseID)
	}
}

func (c *PhysicalBackupController) startStreamHeartbeat(ctx context.Context, userID uuid.UUID) {
	ticker := time.NewTicker(stream_guard.GetDownloadHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.restoreTokenService.RefreshDownloadLock(ctx, userID)
		}
	}
}
