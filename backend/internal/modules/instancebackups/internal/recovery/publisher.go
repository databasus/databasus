package recovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"databasus-backend/internal/engines/postgresql/internaldb"
	"databasus-backend/internal/lib/clock"
)

// maintenanceDatabase is the database publication connects to, because the active database cannot
// be renamed while a session uses it.
const maintenanceDatabase = "postgres"

type PublisherDeps struct {
	Directory Directory
	Journal   *JournalStore
	// Server reaches the active application database.
	Server        internaldb.Connection
	SecretKeyPath string
	Clock         clock.Source
	Logger        *slog.Logger
}

// Publisher installs a prepared recovery: it swaps the staging database in under the active name,
// installs the exact source key and records completion only once both, with the recovery pause,
// are durable. Every step can be repeated, so a restart resumes from the journal.
type Publisher struct {
	directory     Directory
	journal       *JournalStore
	server        internaldb.Connection
	secretKeyPath string
	clock         clock.Source
	logger        *slog.Logger
}

func NewPublisher(deps PublisherDeps) *Publisher {
	return &Publisher{
		directory:     deps.Directory,
		journal:       deps.Journal,
		server:        deps.Server,
		secretKeyPath: deps.SecretKeyPath,
		clock:         deps.Clock,
		logger:        deps.Logger,
	}
}

// Publish finishes the journal's operation, which must be IMPORTED or later. The caller holds the
// lifecycle lock exclusive and has closed every connection to the active database.
func (p *Publisher) Publish(ctx context.Context, journal Journal) (Journal, error) {
	operation := journal.Operation
	if operation.Phase == PhaseCompleted {
		return p.FinishCompleted(ctx, journal)
	}

	if operation.Phase != PhasePublishing {
		operation.Phase = PhasePublishing
		if err := p.journal.Write(journal); err != nil {
			return journal, err
		}
	}

	if err := p.retainPreviousKey(journal); err != nil {
		return journal, err
	}

	if err := p.swapDatabasesOnce(ctx, journal); err != nil {
		return p.returnToImported(journal, err)
	}

	keyContent, err := os.ReadFile(p.directory.getStagedKeyPath(operation.ID))
	if err != nil {
		return journal, fmt.Errorf("read the staged master key: %w", err)
	}

	if err := writeFileDurably(p.secretKeyPath, keyContent); err != nil {
		return journal, fmt.Errorf("install the recovered master key: %w", err)
	}

	if err := p.checkPause(ctx, journal); err != nil {
		return journal, err
	}

	journal = p.recordCompletion(journal, hashBytes(keyContent))
	if err := p.journal.Write(journal); err != nil {
		return journal, err
	}

	p.logger.InfoContext(ctx, "recovery published",
		slog.String("operation_id", operation.ID.String()),
		slog.String("origin", string(operation.Origin)),
		slog.String("archive_sha256", operation.ArchiveSha256),
		slog.String("source_release", operation.SourceRelease))

	return p.FinishCompleted(ctx, journal)
}

// FinishCompleted releases what a completed operation no longer needs: the replaced database and
// the operation's staged and retained files.
func (p *Publisher) FinishCompleted(ctx context.Context, journal Journal) (Journal, error) {
	operation := journal.Operation

	if err := internaldb.DropPreviousDatabase(ctx, p.getMaintenance(), operation.PreviousDatabase); err != nil {
		return journal, err
	}

	if err := p.directory.removeOperationDirectory(operation.ID); err != nil {
		return journal, err
	}

	journal.Operation = nil

	return journal, p.journal.Write(journal)
}

// Discard drops an operation that was not published, leaving the active database and key as they
// were.
func (p *Publisher) Discard(ctx context.Context, journal Journal) (Journal, error) {
	operation := journal.Operation

	if err := internaldb.DropStagingDatabase(ctx, p.getMaintenance(), operation.StagingDatabase); err != nil {
		return journal, err
	}

	if err := p.directory.removeOperationDirectory(operation.ID); err != nil {
		return journal, err
	}

	journal.Operation = nil

	return journal, p.journal.Write(journal)
}

// End discards an operation that was not published and records how it ended, so root admin sees
// it. A nil failure records a cancellation.
func (p *Publisher) End(ctx context.Context, journal Journal, failure *Failure) (Journal, error) {
	operation := *journal.Operation

	journal, err := p.Discard(ctx, journal)
	if err != nil {
		return journal, err
	}

	journal.LastEnded = &EndedRecovery{
		OperationID:   operation.ID,
		Origin:        operation.Origin,
		ArchiveSha256: operation.ArchiveSha256,
		SourceRelease: operation.SourceRelease,
		SnapshotAt:    operation.SnapshotAt,
		CreatedAt:     operation.CreatedAt,
		EndedAt:       p.clock.Now(),
		Failure:       failure,
	}

	return journal, p.journal.Write(journal)
}

// returnToImported records that a failed swap changed nothing: the swap is one transaction, and the
// key is installed only after it.
func (p *Publisher) returnToImported(journal Journal, swapErr error) (Journal, error) {
	journal.Operation.Phase = PhaseImported
	if err := p.journal.Write(journal); err != nil {
		return journal, errors.Join(swapErr, err)
	}

	return journal, fmt.Errorf("%w: %w", ErrPublicationNotStarted, swapErr)
}

// retainPreviousKey keeps a copy of the replaced key until completion. A resumed publication finds
// the copy and leaves it, because the active key may already be the recovered one.
func (p *Publisher) retainPreviousKey(journal Journal) error {
	previousKeyPath := p.directory.getPreviousKeyPath(journal.Operation.ID)
	if _, err := os.Stat(previousKeyPath); err == nil {
		return nil
	}

	content, err := os.ReadFile(p.secretKeyPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("read the replaced master key: %w", err)
	}

	return writeFileDurably(previousKeyPath, content)
}

// swapDatabasesOnce swaps while the staging database still exists. The swap is one transaction,
// so a missing staging database means it already happened.
func (p *Publisher) swapDatabasesOnce(ctx context.Context, journal Journal) error {
	operation := journal.Operation

	stagingNames, err := internaldb.ListDatabases(ctx, p.getMaintenance(), operation.StagingDatabase)
	if err != nil {
		return err
	}

	if !slices.Contains(stagingNames, operation.StagingDatabase) {
		return nil
	}

	return internaldb.SwapDatabases(ctx, p.getMaintenance(), internaldb.SwapSpec{
		Active:   p.server.Database,
		Staging:  operation.StagingDatabase,
		Previous: operation.PreviousDatabase,
	})
}

func (p *Publisher) checkPause(ctx context.Context, journal Journal) error {
	active, err := openDatabaseRepository(ctx, p.server)
	if err != nil {
		return err
	}
	defer active.close()

	pausedBy, err := active.getPause(ctx)
	if err != nil {
		return err
	}

	if pausedBy == nil || *pausedBy != journal.Operation.ID {
		return fmt.Errorf("%w: the active database is not paused by this recovery", ErrStagedStateInvalid)
	}

	return nil
}

func (p *Publisher) recordCompletion(journal Journal, keySha256 string) Journal {
	operation := journal.Operation

	journal.PublicationGeneration++
	journal.PublishedKeySha256 = keySha256

	if operation.Origin == OriginStartup && !journal.IsStartupArchiveConsumed(operation.ArchiveSha256) {
		journal.ConsumedStartupArchives = append(journal.ConsumedStartupArchives, operation.ArchiveSha256)
	}

	journal.LastCompleted = &CompletedRecovery{
		OperationID:           operation.ID,
		Origin:                operation.Origin,
		ArchiveSha256:         operation.ArchiveSha256,
		SourceRelease:         operation.SourceRelease,
		SnapshotAt:            operation.SnapshotAt,
		CreatedAt:             operation.CreatedAt,
		CompletedAt:           p.clock.Now(),
		DiscardedCleanupCount: operation.DiscardedCleanupCount,
	}
	operation.Phase = PhaseCompleted

	return journal
}

func (p *Publisher) getMaintenance() internaldb.Connection {
	return p.server.WithDatabase(maintenanceDatabase)
}
