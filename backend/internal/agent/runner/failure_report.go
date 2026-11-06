package runner

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/lib/agentclient"
)

type failureReport struct {
	VerificationID uuid.UUID
	RestoreResult  *restore.Result
	Message        string
	Logger         *slog.Logger
	Diagnostic     *pgrestore.FailureDiagnostic
	Secrets        []string
	Cause          error
}

func (r *Runner) reportFailure(ctx context.Context, failure failureReport) {
	if r.shouldSkipFailedReport(ctx, failure.VerificationID, failure.Logger) {
		return
	}

	req := agentclient.VerificationReport{
		Status:      agentclient.VerificationReportStatusFAILED,
		FailMessage: &failure.Message,
	}
	if failure.RestoreResult != nil {
		req.PgRestoreExitCode = &failure.RestoreResult.PgRestoreExitCode
		req.RestoreDurationMs = &failure.RestoreResult.DurationMs
	}

	diagnostic := failure.Diagnostic
	if failure.Cause != nil {
		output := failure.Cause.Error()
		isTruncated := false
		if diagnostic != nil {
			output += "\n" + diagnostic.Output
			isTruncated = diagnostic.IsTruncated
		}
		diagnostic = pgrestore.GetFailureDiagnostic(pgrestore.FailureDiagnosticSpec{
			Output: output, Secrets: failure.Secrets, IsTruncated: isTruncated,
		})
	}

	if diagnostic != nil {
		req.FailureDiagnostic = &agentclient.VerificationFailureDiagnostic{
			Output:      diagnostic.Output,
			IsTruncated: diagnostic.IsTruncated,
		}
	}

	r.sendReport(ctx, failure.VerificationID, sanitizeFailureReport(req, failure.Secrets), failure.Logger)
}

func sanitizeFailureReport(req agentclient.VerificationReport, secrets []string) agentclient.VerificationReport {
	if req.Status == agentclient.VerificationReportStatusFAILED {
		if req.FailMessage != nil {
			summary := pgrestore.GetFailureDiagnostic(
				pgrestore.FailureDiagnosticSpec{Output: *req.FailMessage, Secrets: secrets, IsTruncated: false},
			)
			req.FailMessage = nil
			if summary != nil {
				req.FailMessage = &summary.Output
			}
		}

		if req.FailureDiagnostic != nil {
			diagnostic := pgrestore.GetFailureDiagnostic(pgrestore.FailureDiagnosticSpec{
				Output: req.FailureDiagnostic.Output, Secrets: secrets, IsTruncated: req.FailureDiagnostic.IsTruncated,
			})
			req.FailureDiagnostic = nil
			if diagnostic != nil {
				req.FailureDiagnostic = &agentclient.VerificationFailureDiagnostic{
					Output:      diagnostic.Output,
					IsTruncated: diagnostic.IsTruncated,
				}
			}
		}
	}

	return req
}
