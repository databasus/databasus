package verification_runs

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	verification_bridge "databasus-backend/internal/features/verification"
	verification_agents "databasus-backend/internal/features/verification/agents"
	"databasus-backend/internal/modules/verification"
)

type VerificationAgentController struct {
	logger *slog.Logger
}

func (c *VerificationAgentController) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/agent/verifications/:agentId")
	g.Use(verification_agents.RequireAgentAuth())

	g.POST("/claim", c.Claim)
	g.GET("/:id/backup-stream", c.BackupStream)
	g.POST("/:id/report", c.Report)
}

func (c *VerificationAgentController) Claim(ctx *gin.Context) {
	agent, ok := verification_agents.GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	var req ClaimRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assignment, isAssigned, err := verification_bridge.GetModule().
		ClaimVerification(ctx.Request.Context(), *agent, verification.AgentCapacity{MaxCPU: req.Capacity.MaxCPU, MaxRAMMb: req.Capacity.MaxRAMMb, MaxDiskGb: req.Capacity.MaxDiskGb, MaxConcurrentJobs: req.Capacity.MaxConcurrentJobs})
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !isAssigned {
		ctx.Status(http.StatusNoContent)
		return
	}

	ctx.JSON(
		http.StatusOK,
		JobAssignment{
			VerificationID:     assignment.VerificationID,
			BackupID:           assignment.BackupID,
			BackupSizeMb:       assignment.BackupSizeMb,
			MaxContainerDiskMb: assignment.MaxContainerDiskMb,
			TimescaledbVersion: assignment.TimescaledbVersion,
			Database: AssignedDatabase{
				Type:              assignment.Database.Type,
				PostgresqlLogical: &AssignedPostgresql{Version: assignment.Database.PostgresqlLogical.Version},
			},
		},
	)
}

func (c *VerificationAgentController) BackupStream(ctx *gin.Context) {
	agent, ok := verification_agents.GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	reader, err := verification_bridge.GetModule().
		OpenVerificationBackup(ctx.Request.Context(), *agent, verificationID)
	if err != nil {
		ctx.JSON(http.StatusGone, gin.H{"error": err.Error(), "reason": "gone"})
		return
	}

	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			c.logger.ErrorContext(
				ctx.Request.Context(),
				"failed to close backup reader",
				"error", closeErr,
				"verification_id", verificationID,
			)
		}
	}()

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="verification-%s.dump"`, verificationID),
	)

	if _, err := io.Copy(ctx.Writer, reader); err != nil {
		c.logger.ErrorContext(
			ctx.Request.Context(),
			"failed to stream backup to agent",
			"error", err,
			"verification_id", verificationID,
		)
	}
}

func (c *VerificationAgentController) Report(ctx *gin.Context) {
	agent, ok := verification_agents.GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	var req ReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := verification_bridge.GetModule().
		SubmitVerificationReport(ctx.Request.Context(), *agent, verificationID, toVerificationReport(req)); err != nil {
		if errors.Is(err, verification.ErrVerificationGone) {
			ctx.JSON(http.StatusGone, gin.H{"reason": "gone"})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func toVerificationReport(report ReportRequest) verification.VerificationReport {
	stats := make([]verification.ReportTableStat, len(report.TableStats))
	for index, stat := range report.TableStats {
		stats[index] = verification.ReportTableStat{
			SchemaName: stat.SchemaName,
			Name:       stat.Name,
			RowCount:   stat.RowCount,
		}
	}
	return verification.VerificationReport{
		Status:                    verification.VerificationStatus(report.Status),
		PgRestoreExitCode:         report.PgRestoreExitCode,
		FailureKind:               report.FailureKind,
		RestoreDurationMs:         report.RestoreDurationMs,
		VerifyDurationMs:          report.VerifyDurationMs,
		RestoredDatabaseSizeBytes: report.DBSizeBytesAfterRestore,
		TableCount:                report.TableCount,
		SchemaCount:               report.SchemaCount,
		TableStats:                stats,
		FailMessage:               report.FailMessage,
	}
}
