package backups_controllers_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/users"
)

type PhysicalBackupController struct {
	getBackups func() backups.Module
	logger     *slog.Logger
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
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups.GetPhysicalBackupsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.getBackups().ListPhysicalBackups(ctx.Request.Context(), actor, databaseID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *PhysicalBackupController) GetWalStreamLive(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	response, err := c.getBackups().GetWalStreamLiveState(ctx.Request.Context(), actor, databaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *PhysicalBackupController) GetWalStreamStatistics(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups.GetWalStreamStatisticsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.getBackups().GetWalStreamStatistics(ctx.Request.Context(), actor, databaseID, request.Period)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *PhysicalBackupController) TriggerBackup(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups.TriggerBackupRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.requestBackupOfType(ctx.Request.Context(), actor, databaseID, request.Type); err != nil {
		if errors.Is(err, backups.ErrNoExtendablePhysicalChain) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "backup requested"})
}

func (c *PhysicalBackupController) GenerateRestoreToken(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups.GenerateRestoreTokenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := c.getBackups().IssuePhysicalRestoreToken(ctx.Request.Context(), actor, databaseID, request.TargetTime)
	if err != nil {
		if errors.Is(err, backups.ErrUserStreamHeld) {
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

	ctx.JSON(http.StatusOK, backups.GenerateRestoreTokenResponse{
		Token: token,
		URL:   "/api/v1/backups/physical/restore-stream?token=" + token,
	})
}

func (c *PhysicalBackupController) GenerateBackupRestoreToken(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	token, err := c.getBackups().IssuePhysicalBackupRestoreToken(ctx.Request.Context(), actor, backupID)
	if err != nil {
		if errors.Is(err, backups.ErrPhysicalBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		if errors.Is(err, backups.ErrUserStreamHeld) {
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

	ctx.JSON(http.StatusOK, backups.GenerateRestoreTokenResponse{
		Token: token,
		URL:   "/api/v1/backups/physical/restore-stream?token=" + token,
	})
}

func (c *PhysicalBackupController) CancelBackup(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.getBackups().CancelPhysicalBackup(ctx.Request.Context(), actor, backupID); err != nil {
		if errors.Is(err, backups.ErrPhysicalBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *PhysicalBackupController) DeleteBackup(ctx *gin.Context) {
	actor, ok := users.GetActor(ctx.Request.Context())
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.getBackups().DeletePhysicalBackup(ctx.Request.Context(), actor, backupID); err != nil {
		if errors.Is(err, backups.ErrPhysicalBackupNotFound) {
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

	restoreStream, err := c.getBackups().OpenPhysicalRestoreStream(ctx.Request.Context(), token)
	if err != nil {
		if errors.Is(err, backups.ErrUserStreamHeld) {
			ctx.JSON(http.StatusConflict, gin.H{"error": "a download or restore is already in progress for this user"})
			return
		}
		if errors.Is(err, backups.ErrInvalidPhysicalRestoreToken) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired restore token"})
			return
		}
		status, message := mapRestoreStreamError(err)
		ctx.JSON(status, gin.H{"error": message})
		return
	}
	defer restoreStream.Close()
	ctx.Header("Content-Type", "application/x-tar")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="restore-%s.tar"`, restoreStream.DatabaseID))
	if err := restoreStream.Write(ctx.Writer); err != nil {
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
	var gapErr backups.PhysicalWalGapBeforeTargetError
	if errors.As(err, &gapErr) {
		return http.StatusUnprocessableEntity, gapErr.Error(), true
	}

	if errors.Is(err, backups.ErrPhysicalTargetBeforeEarliest) {
		return http.StatusUnprocessableEntity, err.Error(), true
	}

	if errors.Is(err, backups.ErrNoPhysicalChainForRestore) {
		return http.StatusNotFound, err.Error(), true
	}

	return 0, "", false
}

func (c *PhysicalBackupController) requestBackupOfType(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	backupType backups.TriggerBackupType,
) error {
	return c.getBackups().RequestPhysicalBackup(ctx, actor, databaseID, backupType)
}
