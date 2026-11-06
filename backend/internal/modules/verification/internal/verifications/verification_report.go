package verifications

import (
	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/lib/apierror"
)

type ReportTableStat struct {
	SchemaName string
	Name       string
	RowCount   int64
}

type ReportRequest struct {
	Status                    VerificationStatus
	PgRestoreExitCode         *int
	FailureKind               *string
	RestoreDurationMs         *int64
	VerifyDurationMs          *int64
	RestoredDatabaseSizeBytes *int64
	TableCount                *int
	SchemaCount               *int
	TableStats                []ReportTableStat
	FailMessage               *string
	FailureDiagnostic         *pgrestore.FailureDiagnostic
}

func (r *ReportRequest) validateAndSanitizeFailure() error {
	if r.FailureDiagnostic != nil {
		if r.Status != VerificationStatusFailed || len(r.FailureDiagnostic.Output) == 0 ||
			len(r.FailureDiagnostic.Output) > pgrestore.MaxFailureDiagnosticBytes {
			return apierror.NewInvalidFieldsProblem(
				[]apierror.FieldError{
					{Field: "failureDiagnostic.output", Detail: "requires a failed report and 1 to 65536 UTF-8 bytes"},
				},
			)
		}

		r.FailureDiagnostic = pgrestore.GetFailureDiagnostic(pgrestore.FailureDiagnosticSpec{
			Output: r.FailureDiagnostic.Output, IsTruncated: r.FailureDiagnostic.IsTruncated, Secrets: nil,
		})
	}

	if r.FailMessage != nil {
		summary := pgrestore.GetFailureDiagnostic(
			pgrestore.FailureDiagnosticSpec{Output: *r.FailMessage, Secrets: nil, IsTruncated: false},
		)
		r.FailMessage = nil
		if summary != nil {
			r.FailMessage = &summary.Output
		}
	}

	return nil
}
