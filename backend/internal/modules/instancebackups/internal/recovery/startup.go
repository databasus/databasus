package recovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/internaldb"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
)

// stagingHeadroomBytes is the free space a staged archive copy leaves for the journal and logs.
const stagingHeadroomBytes = 64 << 20

type StartupDeps struct {
	Directory     Directory
	Server        internaldb.Connection
	SecretKeyPath string
	DataFolder    string
	ClientTools   pgshared.ClientTools
	Release       string
	Bundle        database.MigrationBundle
	// DumpPath and MasterKeyPath are both empty when startup recovery is not configured.
	DumpPath      string
	MasterKeyPath string
	// NewerSchemaGuidance is the shared next step for a database migrated by a newer version.
	NewerSchemaGuidance string
	// Normalizers end the other modules' recovered work in the staging database.
	Normalizers []StateNormalizer
	Clock       clock.Source
	Logger      *slog.Logger
	InspectDisk func(string) (diskusage.Usage, error)
}

// Startup runs before the application opens its database. It takes the lifecycle lock, finishes a
// publication a previous process left unfinished and runs a configured startup recovery.
type Startup struct {
	directory           Directory
	journal             *JournalStore
	server              internaldb.Connection
	secretKeyPath       string
	dumpPath            string
	masterKeyPath       string
	newerSchemaGuidance string
	clock               clock.Source
	logger              *slog.Logger
	inspectDisk         func(string) (diskusage.Usage, error)
	inspector           *TargetInspector
	preparer            *Preparer
	publisher           *Publisher
}

func NewStartup(deps StartupDeps) *Startup {
	journal := NewJournalStore(deps.Directory)

	return &Startup{
		directory:           deps.Directory,
		journal:             journal,
		server:              deps.Server,
		secretKeyPath:       deps.SecretKeyPath,
		dumpPath:            deps.DumpPath,
		masterKeyPath:       deps.MasterKeyPath,
		newerSchemaGuidance: deps.NewerSchemaGuidance,
		clock:               deps.Clock,
		logger:              deps.Logger,
		inspectDisk:         deps.InspectDisk,
		inspector:           NewTargetInspector(deps.DataFolder, deps.Bundle),
		preparer: NewPreparer(PreparerDeps{
			Server:      deps.Server,
			ClientTools: deps.ClientTools,
			Release:     deps.Release,
			Bundle:      deps.Bundle,
			Normalizers: deps.Normalizers,
			Clock:       deps.Clock,
			Logger:      deps.Logger,
		}),
		publisher: NewPublisher(PublisherDeps{
			Directory:     deps.Directory,
			Journal:       journal,
			Server:        deps.Server,
			SecretKeyPath: deps.SecretKeyPath,
			Clock:         deps.Clock,
			Logger:        deps.Logger,
		}),
	}
}

// OpenForServer returns the lifecycle lock held shared for the server's lifetime. A refused
// configured recovery is logged for the operator and returned, so the server does not start.
func (s *Startup) OpenForServer(ctx context.Context) (*LifecycleLock, error) {
	lock, err := s.openLock()
	if err != nil {
		return nil, err
	}

	err = s.prepareServerStart(ctx, lock)
	if err != nil {
		_ = lock.Close()

		return nil, err
	}

	return lock, nil
}

// OpenForConsole returns the lifecycle lock held shared for a console command. It never runs a
// recovery and refuses while one is unfinished.
func (s *Startup) OpenForConsole() (*LifecycleLock, error) {
	lock, err := s.openLock()
	if err != nil {
		return nil, err
	}

	err = s.openShared(lock)
	if err != nil {
		_ = lock.Close()

		return nil, err
	}

	return lock, nil
}

func (s *Startup) openLock() (*LifecycleLock, error) {
	if err := s.directory.Ensure(); err != nil {
		return nil, err
	}

	return OpenLifecycleLock(s.directory)
}

func (s *Startup) prepareServerStart(ctx context.Context, lock *LifecycleLock) error {
	isExclusive, err := lock.TryExclusive()
	if err != nil {
		return err
	}

	if !isExclusive {
		return s.startBesideAnotherProcess(ctx, lock)
	}

	journal, err := s.journal.Read()
	if err != nil {
		return err
	}

	journal, err = s.resolveUnfinished(ctx, journal)
	if err != nil {
		return err
	}

	if err := s.checkPublishedKey(journal); err != nil {
		return err
	}

	if s.isConfigured() {
		if err := s.runConfigured(ctx, journal); err != nil {
			return err
		}
	}

	isShared, err := lock.TryShared()
	if err != nil {
		return err
	}

	if !isShared {
		return ErrLifecycleBusy
	}

	return nil
}

// startBesideAnotherProcess serves next to a running server or console command. A configured
// archive may only be skipped as consumed, because recovery needs the lock exclusive.
func (s *Startup) startBesideAnotherProcess(ctx context.Context, lock *LifecycleLock) error {
	if s.isConfigured() {
		archiveSha256, err := hashFile(s.dumpPath)
		if err != nil {
			return s.refuse(ctx, newInputRefusal("the Databasus backup file", s.dumpPath, err))
		}

		journal, err := s.journal.Read()
		if err != nil {
			return err
		}

		if !journal.IsStartupArchiveConsumed(archiveSha256) {
			return s.refuse(ctx, &RefusalError{
				Reason:      RefusalNoExclusiveAccess,
				Explanation: "another Databasus server or console command uses this data directory",
				NextStep:    "Stop every other Databasus process that uses this data directory, then restart",
				Cause:       ErrLifecycleBusy,
			})
		}

		s.logSkip(ctx, archiveSha256)
	}

	return s.openShared(lock)
}

func (s *Startup) openShared(lock *LifecycleLock) error {
	isShared, err := lock.TryShared()
	if err != nil {
		return err
	}

	if !isShared {
		return ErrLifecycleBusy
	}

	journal, err := s.journal.Read()
	if err != nil {
		return err
	}

	if journal.IsPublicationUnfinished() {
		return fmt.Errorf("%w: stop every Databasus process and start the server alone to finish it",
			ErrPublicationUnfinished)
	}

	return s.checkPublishedKey(journal)
}

// resolveUnfinished finishes an operation that reached publication or whose apply was accepted,
// and discards any other, which never changed the active database or key.
//
// The journal records an operation before its staging database is created, and a discard drops
// that database before it clears the operation, so no staging database outlives its record.
func (s *Startup) resolveUnfinished(ctx context.Context, journal Journal) (Journal, error) {
	if journal.Operation == nil {
		return journal, nil
	}

	if journal.IsPublicationUnfinished() {
		s.logger.InfoContext(ctx, "finishing an interrupted recovery publication",
			slog.String("operation_id", journal.Operation.ID.String()),
			slog.String("phase", string(journal.Operation.Phase)))

		return s.publisher.Publish(ctx, journal)
	}

	s.logger.InfoContext(ctx, "discarding an unfinished recovery preparation; the current installation is unchanged",
		slog.String("operation_id", journal.Operation.ID.String()),
		slog.String("origin", string(journal.Operation.Origin)))

	if journal.Operation.Origin == OriginUpload {
		return s.publisher.End(ctx, journal, new(FailureInterrupted))
	}

	return s.publisher.Discard(ctx, journal)
}

func (s *Startup) checkPublishedKey(journal Journal) error {
	if journal.PublishedKeySha256 == "" {
		return nil
	}

	keyContent, err := os.ReadFile(s.secretKeyPath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPublishedKeyMismatch, err)
	}

	if hashBytes(keyContent) != journal.PublishedKeySha256 {
		return fmt.Errorf("%w: restore the key file the recovery installed at %s",
			ErrPublishedKeyMismatch, s.secretKeyPath)
	}

	return nil
}

func (s *Startup) isConfigured() bool {
	return s.dumpPath != "" && s.masterKeyPath != ""
}

func (s *Startup) runConfigured(ctx context.Context, journal Journal) error {
	archiveSha256, err := hashFile(s.dumpPath)
	if err != nil {
		return s.refuse(ctx, newInputRefusal("the Databasus backup file", s.dumpPath, err))
	}

	if journal.IsStartupArchiveConsumed(archiveSha256) {
		s.logSkip(ctx, archiveSha256)

		return nil
	}

	keyContent, err := os.ReadFile(s.masterKeyPath)
	if err != nil {
		return s.refuse(ctx, newInputRefusal("the master key file", s.masterKeyPath, err))
	}

	masterKey, err := crypto.NewMasterKey(keyContent)
	if err != nil {
		return s.refuse(ctx, newInvalidKeyRefusal(err))
	}

	if err := s.inspectTarget(ctx); err != nil {
		return s.refuse(ctx, err)
	}

	journal, err = s.prepare(ctx, journal, archiveSha256, keyContent, masterKey)
	if err != nil {
		return s.refuse(ctx, err)
	}

	_, err = s.publishPrepared(ctx, journal)
	if errors.Is(err, ErrPublicationUnfinished) {
		s.logger.ErrorContext(ctx, "startup recovery publication did not finish",
			slog.String("current_installation", "not started; the next start finishes the publication"),
			slog.String("next_step", "Fix the error, then restart Databasus with no other Databasus process"),
			slog.Any("error", err))

		return err
	}

	if err != nil {
		return s.refuse(ctx, err)
	}

	return nil
}

func (s *Startup) inspectTarget(ctx context.Context) error {
	target, err := openDatabaseRepository(ctx, s.server)
	if err != nil {
		return newUnknownStateRefusal(err)
	}
	defer target.close()

	return s.inspector.Inspect(ctx, target, s.newerSchemaGuidance)
}

// prepare stages the archive and key, authenticates the archive and imports it into a paused
// staging database. A failure discards everything it staged.
func (s *Startup) prepare(
	ctx context.Context,
	journal Journal,
	archiveSha256 string,
	keyContent []byte,
	masterKey *crypto.MasterKey,
) (Journal, error) {
	if err := s.checkDiskSpace(); err != nil {
		return journal, err
	}

	operationID := uuid.New()
	stagingDatabase, previousDatabase := getDatabaseNames(s.server.Database, operationID)
	journal.Operation = &Operation{
		ID:                    operationID,
		Origin:                OriginStartup,
		Phase:                 PhasePreparing,
		ArchiveSha256:         archiveSha256,
		SourceRelease:         "",
		SnapshotAt:            nil,
		StagingDatabase:       stagingDatabase,
		PreviousDatabase:      previousDatabase,
		IsApplyAccepted:       false,
		CreatedAt:             s.clock.Now(),
		DiscardedCleanupCount: 0,
	}

	if err := s.journal.Write(journal); err != nil {
		return journal, err
	}

	journal, prepareErr := s.stageAndImport(ctx, journal, keyContent, masterKey)
	if prepareErr != nil {
		journal, err := s.publisher.Discard(context.WithoutCancel(ctx), journal)

		return journal, errors.Join(prepareErr, err)
	}

	return journal, nil
}

func (s *Startup) stageAndImport(
	ctx context.Context,
	journal Journal,
	keyContent []byte,
	masterKey *crypto.MasterKey,
) (Journal, error) {
	operation := journal.Operation

	if err := s.directory.createOperationDirectory(operation.ID); err != nil {
		return journal, err
	}

	stagedSha256, err := copyFileDurably(s.dumpPath, s.directory.getStagedArchivePath(operation.ID))
	if err != nil {
		return journal, newInputRefusal("the Databasus backup file", s.dumpPath, err)
	}

	if stagedSha256 != operation.ArchiveSha256 {
		return journal, newInputRefusal("the Databasus backup file", s.dumpPath, ErrArchiveChanged)
	}

	if err := writeFileDurably(s.directory.getStagedKeyPath(operation.ID), keyContent); err != nil {
		return journal, err
	}

	if err := s.preparer.Prepare(
		ctx,
		operation,
		s.directory.getStagedArchivePath(operation.ID),
		masterKey,
	); err != nil {
		return journal, err
	}

	operation.Phase = PhaseImported

	return journal, s.journal.Write(journal)
}

// publishPrepared rechecks the target under the exclusive lock, so user state created while the
// archive was imported is never replaced, then publishes.
func (s *Startup) publishPrepared(ctx context.Context, journal Journal) (Journal, error) {
	if err := s.inspectTarget(ctx); err != nil {
		journal, discardErr := s.publisher.Discard(context.WithoutCancel(ctx), journal)

		return journal, errors.Join(err, discardErr)
	}

	journal, err := s.publisher.Publish(ctx, journal)
	if errors.Is(err, ErrPublicationNotStarted) {
		journal, discardErr := s.publisher.Discard(context.WithoutCancel(ctx), journal)

		return journal, errors.Join(newImportRefusal(err), discardErr)
	}

	if err != nil {
		return journal, fmt.Errorf("%w: %w", ErrPublicationUnfinished, err)
	}

	return journal, nil
}

func (s *Startup) checkDiskSpace() error {
	archiveInfo, err := os.Stat(s.dumpPath)
	if err != nil {
		return newInputRefusal("the Databasus backup file", s.dumpPath, err)
	}

	usage, err := s.inspectDisk(s.directory.root)
	if err != nil {
		return fmt.Errorf("check free space for the staged recovery: %w", err)
	}

	neededBytes := archiveInfo.Size() + stagingHeadroomBytes
	if usage.FreeBytes < neededBytes {
		return &RefusalError{
			Reason: RefusalInsufficientDisk,
			Explanation: fmt.Sprintf("the data volume has %d MB free, and staging the archive needs %d MB",
				usage.FreeBytes>>20, neededBytes>>20),
			NextStep: "Free space in the data volume, then restart",
			Cause:    nil,
		}
	}

	return nil
}

func (s *Startup) logSkip(ctx context.Context, archiveSha256 string) {
	s.logger.InfoContext(ctx, "startup recovery skipped an archive it already recovered",
		slog.String("archive_sha256", archiveSha256),
		slog.String("current_installation", "starts normally from its current state"),
		slog.String("next_step", "none needed; the recovery settings may stay configured"))
}

// refuse logs a configured startup recovery that did not run. Every refusal leaves the active
// database and key unchanged, which the log says.
func (s *Startup) refuse(ctx context.Context, err error) error {
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		refusal = newImportRefusal(err)
	}

	s.logger.ErrorContext(ctx, "startup recovery did not run",
		slog.String("reason", string(refusal.Reason)),
		slog.String("explanation", refusal.Explanation),
		slog.String("current_installation", "unchanged"),
		slog.String("next_step", refusal.NextStep),
		slog.Any("error", err))

	return refusal
}

func newInputRefusal(input, path string, err error) *RefusalError {
	return &RefusalError{
		Reason:      RefusalInputUnreadable,
		Explanation: input + " at " + path + " cannot be read",
		NextStep: "Put the file in a volume mounted into the container, make it readable by Databasus, " +
			"check RECOVERY_DUMP_PATH and RECOVERY_MASTER_KEY_PATH, then restart",
		Cause: err,
	}
}

func newInvalidKeyRefusal(err error) *RefusalError {
	return &RefusalError{
		Reason:      RefusalInvalidMasterKey,
		Explanation: "the master key file does not decrypt this Databasus backup",
		NextStep: "Supply the unchanged master key file saved from the installation that made the backup, " +
			"then restart",
		Cause: err,
	}
}

func newArchiveRefusal(err error) *RefusalError {
	if errors.Is(err, archive.ErrMasterKeyMismatch) {
		return newInvalidKeyRefusal(err)
	}

	return &RefusalError{
		Reason:      RefusalInvalidArchive,
		Explanation: "the file is not a complete, unmodified Databasus backup",
		NextStep:    "Download the Databasus backup again or choose another one, then restart",
		Cause:       err,
	}
}

func newImportRefusal(err error) *RefusalError {
	return &RefusalError{
		Reason:      RefusalImportFailed,
		Explanation: "the Databasus backup could not be imported and published",
		NextStep:    "Check the error, fix its cause, then restart; the archive was not marked as recovered",
		Cause:       err,
	}
}
