package recovery

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Phase is how far one recovery operation got. Publication resumes from PUBLISHING at the next
// start; an earlier phase is discarded unless its apply was accepted.
type Phase string

const (
	PhasePreparing  Phase = "PREPARING"
	PhaseImported   Phase = "IMPORTED"
	PhasePublishing Phase = "PUBLISHING"
	PhaseCompleted  Phase = "COMPLETED"
)

// Origin is where a recovery's archive came from.
type Origin string

const (
	OriginStartup Origin = "STARTUP"
	OriginUpload  Origin = "UPLOAD"
)

// Operation is the one recovery in progress. It holds no key bytes or identity proofs: the staged
// key lives in its own file of the operation directory.
type Operation struct {
	ID            uuid.UUID `json:"id"`
	Origin        Origin    `json:"origin"`
	Phase         Phase     `json:"phase"`
	ArchiveSha256 string    `json:"archiveSha256"`
	SourceRelease string    `json:"sourceRelease,omitempty"`
	// SnapshotAt is set once the archive is authenticated.
	SnapshotAt       *time.Time `json:"snapshotAt,omitempty"`
	StagingDatabase  string     `json:"stagingDatabase"`
	PreviousDatabase string     `json:"previousDatabase"`
	// IsApplyAccepted makes a restart finish the operation instead of discarding it.
	IsApplyAccepted bool      `json:"isApplyAccepted"`
	CreatedAt       time.Time `json:"createdAt"`
	// ExpiresAt is set when an uploaded recovery becomes ready; without an accepted apply by then it
	// is discarded.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	// DiscardedCleanupCount counts the source's unfinished file writes and deletions that
	// normalization dropped instead of replaying.
	DiscardedCleanupCount int `json:"discardedCleanupCount"`
}

// CompletedRecovery is the safe record of the latest successful publication.
type CompletedRecovery struct {
	OperationID   uuid.UUID  `json:"operationId"`
	Origin        Origin     `json:"origin"`
	ArchiveSha256 string     `json:"archiveSha256"`
	SourceRelease string     `json:"sourceRelease"`
	SnapshotAt    *time.Time `json:"snapshotAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	CompletedAt   time.Time  `json:"completedAt"`
	// DiscardedCleanupCount is carried over from the operation, for root admin's cleanup notice.
	DiscardedCleanupCount int `json:"discardedCleanupCount"`
}

// EndedRecovery is the safe record of the latest operation that ended without publication, so root
// admin sees why an upload failed or that it was cancelled.
type EndedRecovery struct {
	OperationID   uuid.UUID  `json:"operationId"`
	Origin        Origin     `json:"origin"`
	ArchiveSha256 string     `json:"archiveSha256"`
	SourceRelease string     `json:"sourceRelease"`
	SnapshotAt    *time.Time `json:"snapshotAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	EndedAt       time.Time  `json:"endedAt"`
	// Failure is nil for a cancelled operation.
	Failure *Failure `json:"failure,omitempty"`
}

// Journal is the recovery state kept outside the internal database, so it survives the
// replacement of that database by a snapshot.
type Journal struct {
	// PublicationGeneration counts completed publications.
	PublicationGeneration int64 `json:"publicationGeneration"`
	// PublishedKeySha256 identifies the key the latest publication installed with its database.
	PublishedKeySha256 string `json:"publishedKeySha256,omitempty"`
	// ConsumedStartupArchives are the SHA-256 fingerprints of every archive a startup recovery
	// published. A later UI import does not remove them.
	ConsumedStartupArchives []string           `json:"consumedStartupArchives"`
	Operation               *Operation         `json:"operation,omitempty"`
	LastCompleted           *CompletedRecovery `json:"lastCompleted,omitempty"`
	LastEnded               *EndedRecovery     `json:"lastEnded,omitempty"`
}

func (j Journal) IsStartupArchiveConsumed(archiveSha256 string) bool {
	return slices.Contains(j.ConsumedStartupArchives, archiveSha256)
}

// IsPublicationUnfinished is true while an operation must be finished before the application may
// open its database.
func (j Journal) IsPublicationUnfinished() bool {
	if j.Operation == nil {
		return false
	}

	return j.Operation.Phase == PhasePublishing || j.Operation.Phase == PhaseCompleted || j.Operation.IsApplyAccepted
}
