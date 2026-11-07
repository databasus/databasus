package agentapi

import (
	"context"
	"fmt"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/modules/verification/internal/agents"
	"databasus-backend/internal/modules/verification/internal/verifications"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Agents        *agents.Service
	Verifications *verifications.Service
}

type Handler struct {
	agents        *agents.Service
	verifications *verifications.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{agents: deps.Agents, verifications: deps.Verifications}
}

func (h *Handler) RecordHeartbeat(
	ctx context.Context,
	request RecordHeartbeatRequestObject,
) (RecordHeartbeatResponseObject, error) {
	heartbeat, err := h.verifications.RecordHeartbeat(
		ctx,
		getAgent(ctx),
		verifications.Heartbeat{
			Capacity: agents.AgentCapacity{
				MaxCPU:            request.Body.MaxCPU,
				MaxRAMMb:          request.Body.MaxRAMMb,
				MaxDiskGb:         request.Body.MaxDiskGb,
				MaxConcurrentJobs: request.Body.MaxConcurrentJobs,
			},
			CurrentVerificationIDs: request.Body.CurrentVerificationIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	return RecordHeartbeat200JSONResponse{
		LastSeenAt:           heartbeat.LastSeenAt,
		AbortVerificationIDs: heartbeat.AbortVerificationIDs,
	}, nil
}

func (h *Handler) ClaimVerification(
	ctx context.Context,
	request ClaimVerificationRequestObject,
) (ClaimVerificationResponseObject, error) {
	assignment, err := h.verifications.ClaimVerification(
		ctx,
		getAgent(ctx),
		verifications.AgentCapacity{
			MaxCPU: request.Body.Capacity.MaxCPU, MaxRAMMb: request.Body.Capacity.MaxRAMMb,
			MaxDiskGb: request.Body.Capacity.MaxDiskGb, MaxConcurrentJobs: request.Body.Capacity.MaxConcurrentJobs,
		},
	)
	if err != nil {
		return nil, err
	}
	if assignment == nil {
		return ClaimVerification204Response{}, nil
	}
	return ClaimVerification200JSONResponse{
		VerificationID:     assignment.VerificationID,
		BackupID:           assignment.BackupID,
		BackupSizeMb:       assignment.BackupSizeMb,
		MaxContainerDiskMb: assignment.MaxContainerDiskMb,
		TimescaledbVersion: assignment.TimescaledbVersion,
		Database: AssignedDatabase{
			Type:              string(assignment.Database.Type),
			PostgresqlLogical: &AssignedPostgresql{Version: string(assignment.Database.PostgresqlLogical.Version)},
		},
	}, nil
}

func (h *Handler) StreamVerificationBackup(
	ctx context.Context,
	request StreamVerificationBackupRequestObject,
) (StreamVerificationBackupResponseObject, error) {
	stream, err := h.verifications.GetBackupFile(
		ctx,
		getAgent(ctx),
		request.VerificationId,
	)
	if err != nil {
		return nil, err
	}
	return StreamVerificationBackup200ApplicationoctetStreamResponse{
		Body: stream,
		Headers: StreamVerificationBackup200ResponseHeaders{
			ContentDisposition: fmt.Sprintf(`attachment; filename="verification-%s.dump"`, request.VerificationId),
		},
	}, nil
}

func (h *Handler) ReportVerification(
	ctx context.Context,
	request ReportVerificationRequestObject,
) (ReportVerificationResponseObject, error) {
	err := h.verifications.SubmitReport(
		ctx,
		getAgent(ctx),
		request.VerificationId,
		toReport(*request.Body),
	)
	return ReportVerification204Response{}, err
}

func toReport(report VerificationReport) *verifications.ReportRequest {
	tables := make([]verifications.ReportTableStat, len(report.TableStats))
	for index, table := range report.TableStats {
		tables[index] = verifications.ReportTableStat{
			SchemaName: table.SchemaName,
			Name:       table.Name,
			RowCount:   table.RowCount,
		}
	}

	var diagnostic *pgrestore.FailureDiagnostic
	if report.FailureDiagnostic != nil {
		diagnostic = &pgrestore.FailureDiagnostic{
			Output:      report.FailureDiagnostic.Output,
			IsTruncated: report.FailureDiagnostic.IsTruncated,
		}
	}

	return &verifications.ReportRequest{
		Status:                    verifications.VerificationStatus(report.Status),
		PgRestoreExitCode:         report.PgRestoreExitCode,
		FailureKind:               report.FailureKind,
		RestoreDurationMs:         report.RestoreDurationMs,
		VerifyDurationMs:          report.VerifyDurationMs,
		RestoredDatabaseSizeBytes: report.DBSizeBytesAfterRestore,
		TableCount:                report.TableCount,
		SchemaCount:               report.SchemaCount,
		TableStats:                tables,
		FailMessage:               report.FailMessage,
		FailureDiagnostic:         diagnostic,
	}
}
