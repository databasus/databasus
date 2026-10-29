package api

type VerificationStatus string

const (
	VerificationStatusCompleted VerificationStatus = "COMPLETED"
	VerificationStatusFailed    VerificationStatus = "FAILED"
)

type FailureType string

// FailureKindDiskLimitExceeded tells the backend the restore exceeded its
// server-computed per-job disk budget — a terminal verdict, distinct from the
// retryable nil-exit-code path.
const FailureKindDiskLimitExceeded FailureType = "DISK_LIMIT_EXCEEDED"
