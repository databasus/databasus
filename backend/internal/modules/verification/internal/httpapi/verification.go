package httpapi

import "databasus-backend/internal/modules/verification/internal/verifications"

func fromVerification(verification *verifications.Verification) Verification {
	return Verification{
		AgentId:                   verification.AgentID,
		AttemptCount:              verification.AttemptCount,
		BackupId:                  verification.BackupID,
		CreatedAt:                 verification.CreatedAt,
		DatabaseId:                verification.DatabaseID,
		FailMessage:               verification.FailMessage,
		FinishedAt:                verification.FinishedAt,
		Id:                        verification.ID,
		PgRestoreExitCode:         verification.PgRestoreExitCode,
		RestoreDurationMs:         verification.RestoreDurationMs,
		RestoredDatabaseSizeBytes: verification.RestoredDatabaseSizeBytes,
		SchemaCount:               verification.SchemaCount,
		StartedAt:                 verification.StartedAt,
		Status:                    VerificationStatus(verification.Status),
		TableCount:                verification.TableCount,
		Trigger:                   VerificationTrigger(verification.Trigger),
		VerifyDurationMs:          verification.VerifyDurationMs,
	}
}

func fromVerifications(verifications []*verifications.Verification) []Verification {
	items := make([]Verification, len(verifications))
	for index, verification := range verifications {
		items[index] = fromVerification(verification)
	}
	return items
}

func fromDetails(verification *verifications.Verification) VerificationDetails {
	stats := make([]VerificationTableStat, len(verification.TableStats))
	for index, stat := range verification.TableStats {
		stats[index] = VerificationTableStat{SchemaName: stat.SchemaName, Name: stat.Name, RowCount: stat.RowCount}
	}

	var diagnostic *VerificationFailureDiagnostic
	if verification.FailureDiagnosticOutput != nil {
		diagnostic = &VerificationFailureDiagnostic{
			Output:      *verification.FailureDiagnosticOutput,
			IsTruncated: verification.IsFailureDiagnosticTruncated,
		}
	}

	return VerificationDetails{
		FailureDiagnostic:         diagnostic,
		AgentId:                   verification.AgentID,
		AttemptCount:              verification.AttemptCount,
		BackupId:                  verification.BackupID,
		CreatedAt:                 verification.CreatedAt,
		DatabaseId:                verification.DatabaseID,
		FailMessage:               verification.FailMessage,
		FinishedAt:                verification.FinishedAt,
		Id:                        verification.ID,
		PgRestoreExitCode:         verification.PgRestoreExitCode,
		RestoreDurationMs:         verification.RestoreDurationMs,
		RestoredDatabaseSizeBytes: verification.RestoredDatabaseSizeBytes,
		SchemaCount:               verification.SchemaCount,
		StartedAt:                 verification.StartedAt,
		Status:                    VerificationStatus(verification.Status),
		TableCount:                verification.TableCount,
		Trigger:                   VerificationTrigger(verification.Trigger),
		VerifyDurationMs:          verification.VerifyDurationMs,
		TableStats:                stats,
	}
}
