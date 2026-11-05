package verifications

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
}
