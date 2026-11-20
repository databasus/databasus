package recovery

import (
	"time"

	"github.com/google/uuid"
)

// Status is where a recovery is, as root admin sees it.
type Status string

const (
	StatusValidating Status = "VALIDATING"
	StatusReady      Status = "READY"
	StatusApplying   Status = "APPLYING"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
	StatusCancelled  Status = "CANCELLED"
)

// Failure is the safe reason a recovery failed.
type Failure string

const (
	FailureArchiveInvalid        Failure = "ARCHIVE_INVALID"
	FailureArchiveIncomplete     Failure = "ARCHIVE_INCOMPLETE"
	FailureMasterKeyInvalid      Failure = "MASTER_KEY_INVALID"
	FailureReleaseMismatch       Failure = "RELEASE_MISMATCH"
	FailureSchemaIncompatible    Failure = "SCHEMA_INCOMPATIBLE"
	FailureInsufficientDiskSpace Failure = "INSUFFICIENT_DISK_SPACE"
	FailureImportFailed          Failure = "IMPORT_FAILED"
	FailureExpired               Failure = "EXPIRED"
	FailureQuiescenceFailed      Failure = "QUIESCENCE_FAILED"
	FailureInterrupted           Failure = "INTERRUPTED"
)

// Recovery is one recovery of the local journal with only its safe fields.
type Recovery struct {
	ID                    uuid.UUID
	Status                Status
	Origin                Origin
	ArchiveSha256         string
	RunningRelease        string
	SourceRelease         *string
	SnapshotAt            *time.Time
	CreatedAt             time.Time
	ExpiresAt             *time.Time
	CompletedAt           *time.Time
	Failure               *Failure
	DiscardedCleanupCount int
}

// findRecovery looks the recovery up among the operation in progress, the latest ended one and
// the latest completed one, the only recoveries the journal keeps.
func findRecovery(
	journal Journal,
	recoveryID uuid.UUID,
	runningRelease string,
) (Recovery, bool) {
	switch {
	case journal.Operation != nil && journal.Operation.ID == recoveryID:
		return newOperationRecovery(*journal.Operation, runningRelease), true
	case journal.LastEnded != nil && journal.LastEnded.OperationID == recoveryID:
		return newEndedRecovery(*journal.LastEnded, runningRelease), true
	case journal.LastCompleted != nil && journal.LastCompleted.OperationID == recoveryID:
		return newCompletedRecovery(*journal.LastCompleted, runningRelease), true
	default:
		return Recovery{}, false
	}
}

func newOperationRecovery(operation Operation, runningRelease string) Recovery {
	status := StatusValidating

	switch operation.Phase {
	case PhasePreparing:
		status = StatusValidating
	case PhaseImported:
		status = StatusReady
		if operation.IsApplyAccepted {
			status = StatusApplying
		}
	case PhasePublishing, PhaseCompleted:
		status = StatusApplying
	}

	return Recovery{
		ID:                    operation.ID,
		Status:                status,
		Origin:                operation.Origin,
		ArchiveSha256:         operation.ArchiveSha256,
		RunningRelease:        runningRelease,
		SourceRelease:         getOptionalRelease(operation.SourceRelease),
		SnapshotAt:            operation.SnapshotAt,
		CreatedAt:             operation.CreatedAt,
		ExpiresAt:             operation.ExpiresAt,
		CompletedAt:           nil,
		Failure:               nil,
		DiscardedCleanupCount: operation.DiscardedCleanupCount,
	}
}

func newEndedRecovery(ended EndedRecovery, runningRelease string) Recovery {
	status := StatusFailed
	if ended.Failure == nil {
		status = StatusCancelled
	}

	return Recovery{
		ID:                    ended.OperationID,
		Status:                status,
		Origin:                ended.Origin,
		ArchiveSha256:         ended.ArchiveSha256,
		RunningRelease:        runningRelease,
		SourceRelease:         getOptionalRelease(ended.SourceRelease),
		SnapshotAt:            ended.SnapshotAt,
		CreatedAt:             ended.CreatedAt,
		ExpiresAt:             nil,
		CompletedAt:           nil,
		Failure:               ended.Failure,
		DiscardedCleanupCount: 0,
	}
}

func newCompletedRecovery(completed CompletedRecovery, runningRelease string) Recovery {
	completedAt := completed.CompletedAt

	return Recovery{
		ID:                    completed.OperationID,
		Status:                StatusCompleted,
		Origin:                completed.Origin,
		ArchiveSha256:         completed.ArchiveSha256,
		RunningRelease:        runningRelease,
		SourceRelease:         getOptionalRelease(completed.SourceRelease),
		SnapshotAt:            completed.SnapshotAt,
		CreatedAt:             completed.CreatedAt,
		ExpiresAt:             nil,
		CompletedAt:           &completedAt,
		Failure:               nil,
		DiscardedCleanupCount: completed.DiscardedCleanupCount,
	}
}

func getOptionalRelease(release string) *string {
	if release == "" {
		return nil
	}

	return &release
}
