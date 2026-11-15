package recovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/internaldb"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
	"databasus-backend/internal/modules/users"
)

const (
	// PreparationJobName is the job that prepares uploaded recoveries and expires unapplied ones.
	PreparationJobName = "instancebackups_recovery_preparation"
	// PreparationJobInterval bounds how late an expiry is noticed; an upload wakes the job at once.
	PreparationJobInterval = 30 * time.Second
	// preparedLifetime is how long a prepared upload may wait for apply.
	preparedLifetime = time.Hour
)

type UploadServiceDeps struct {
	Directory       Directory
	Journal         *JournalStore
	Server          internaldb.Connection
	Preparer        *Preparer
	Publisher       *Publisher
	Users           users.Module
	MaxArchiveBytes int64
	RunningRelease  string
	Clock           clock.Source
	Logger          *slog.Logger
	InspectDisk     func(string) (diskusage.Usage, error)
}

// UploadService receives a Databasus backup file and its source key from root admin, prepares it
// in the background and lets root admin cancel it before apply. The journal admits one operation
// at a time, so an upload never runs beside another recovery.
type UploadService struct {
	directory       Directory
	journal         *JournalStore
	server          internaldb.Connection
	preparer        *Preparer
	publisher       *Publisher
	users           users.Module
	maxArchiveBytes int64
	runningRelease  string
	clock           clock.Source
	logger          *slog.Logger
	inspectDisk     func(string) (diskusage.Usage, error)
	// preparationMutex is held for a whole preparation, so a cancellation waits until the import it
	// stopped has returned before it releases the staging database.
	preparationMutex sync.Mutex
	cancellations    *background.CancellationRegistry
	// cancelRequests holds the operations a cancellation asked to stop, so a preparation that
	// registers its cancel function after that request still stops at once.
	cancelRequests sync.Map
	wakeSignal     chan struct{}
}

func NewUploadService(deps UploadServiceDeps) *UploadService {
	return &UploadService{
		directory:        deps.Directory,
		journal:          deps.Journal,
		server:           deps.Server,
		preparer:         deps.Preparer,
		publisher:        deps.Publisher,
		users:            deps.Users,
		maxArchiveBytes:  deps.MaxArchiveBytes,
		runningRelease:   deps.RunningRelease,
		clock:            deps.Clock,
		logger:           deps.Logger,
		inspectDisk:      deps.InspectDisk,
		preparationMutex: sync.Mutex{},
		cancellations:    background.NewCancellationRegistry(),
		cancelRequests:   sync.Map{},
		wakeSignal:       make(chan struct{}, 1),
	}
}

// Create stages the uploaded archive and key and answers with the recovery validating. Root admin
// is checked before any byte is read; a refused or interrupted upload leaves nothing staged.
func (s *UploadService) Create(ctx context.Context, actor users.Actor, body *multipart.Reader) (Recovery, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return Recovery{}, err
	}

	return s.receive(ctx, body)
}

// Cancel stops a preparation that runs, waits for it to return and releases the staged archive,
// key and staging database. An accepted apply can no longer be cancelled.
func (s *UploadService) Cancel(ctx context.Context, actor users.Actor, recoveryID uuid.UUID) (Recovery, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return Recovery{}, err
	}

	return s.cancel(ctx, recoveryID)
}

// RunPass prepares a staged upload or expires a prepared one that waited too long for apply.
func (s *UploadService) RunPass(ctx context.Context, now time.Time) error {
	journal, err := s.journal.Read()
	if err != nil {
		return err
	}

	operation := journal.Operation
	if operation == nil || operation.Origin != OriginUpload || journal.IsPublicationUnfinished() {
		return nil
	}

	switch {
	case operation.Phase == PhasePreparing && operation.ArchiveSha256 != "":
		return s.prepare(ctx, operation.ID)
	case operation.Phase == PhaseImported && operation.ExpiresAt != nil && !now.Before(*operation.ExpiresAt):
		return s.expire(ctx, operation.ID)
	default:
		return nil
	}
}

// GetWakeSignal starts a preparation pass as soon as an upload is staged.
func (s *UploadService) GetWakeSignal() <-chan struct{} {
	return s.wakeSignal
}

func (s *UploadService) receive(ctx context.Context, body *multipart.Reader) (Recovery, error) {
	operationID, err := s.reserve()
	if err != nil {
		return Recovery{}, err
	}

	received, err := newUploadReceiver(uploadReceiverSpec{
		body:              body,
		archivePath:       s.directory.getStagedArchivePath(operationID),
		maxArchiveBytes:   s.maxArchiveBytes,
		isSecureTransport: httpserver.IsSecureTransport(ctx),
		inspectDisk:       s.inspectDisk,
		diskPath:          s.directory.root,
	}).receive()
	if err == nil {
		err = writeFileDurably(s.directory.getStagedKeyPath(operationID), received.keyContent)
	}

	if err != nil {
		s.release(ctx, operationID)

		return Recovery{}, err
	}

	createdRecovery, err := s.recordReceived(operationID, received.archiveSha256)
	if err != nil {
		s.release(ctx, operationID)

		return Recovery{}, err
	}

	s.logger.InfoContext(ctx, "recovery upload staged",
		slog.String("operation_id", operationID.String()),
		slog.String("archive_sha256", received.archiveSha256))
	s.wake()

	return createdRecovery, nil
}

func (s *UploadService) cancel(ctx context.Context, recoveryID uuid.UUID) (Recovery, error) {
	s.cancelRequests.Store(recoveryID, true)
	defer s.cancelRequests.Delete(recoveryID)

	s.cancellations.Cancel(recoveryID)

	s.preparationMutex.Lock()
	defer s.preparationMutex.Unlock()

	unlock, err := s.journal.Lock()
	if err != nil {
		return Recovery{}, err
	}
	defer unlock()

	journal, err := s.journal.Read()
	if err != nil {
		return Recovery{}, err
	}

	if journal.Operation == nil || journal.Operation.ID != recoveryID {
		return s.getEndedForCancel(journal, recoveryID)
	}

	if journal.IsPublicationUnfinished() {
		return Recovery{}, ErrRecoveryNotCancellable
	}

	journal, err = s.publisher.End(context.WithoutCancel(ctx), journal, nil)
	if err != nil {
		return Recovery{}, err
	}

	s.logger.InfoContext(ctx, "recovery cancelled; the current installation is unchanged",
		slog.String("operation_id", recoveryID.String()))

	return newEndedRecovery(*journal.LastEnded, s.runningRelease), nil
}

// reserve records the upload's operation before any byte is staged, so a second upload is refused
// and a restart removes a partial file.
func (s *UploadService) reserve() (uuid.UUID, error) {
	unlock, err := s.journal.Lock()
	if err != nil {
		return uuid.Nil, err
	}
	defer unlock()

	journal, err := s.journal.Read()
	if err != nil {
		return uuid.Nil, err
	}

	if journal.Operation != nil {
		return uuid.Nil, ErrRecoveryAlreadyRunning
	}

	operationID := uuid.New()
	stagingDatabase, previousDatabase := getDatabaseNames(s.server.Database, operationID)
	journal.Operation = &Operation{
		ID:                    operationID,
		Origin:                OriginUpload,
		Phase:                 PhasePreparing,
		ArchiveSha256:         "",
		SourceRelease:         "",
		SnapshotAt:            nil,
		StagingDatabase:       stagingDatabase,
		PreviousDatabase:      previousDatabase,
		IsApplyAccepted:       false,
		CreatedAt:             s.clock.Now(),
		ExpiresAt:             nil,
		DiscardedCleanupCount: 0,
	}

	if err := s.directory.createOperationDirectory(operationID); err != nil {
		return uuid.Nil, err
	}

	if err := s.journal.Write(journal); err != nil {
		return uuid.Nil, errors.Join(err, s.directory.removeOperationDirectory(operationID))
	}

	return operationID, nil
}

func (s *UploadService) recordReceived(operationID uuid.UUID, archiveSha256 string) (Recovery, error) {
	unlock, err := s.journal.Lock()
	if err != nil {
		return Recovery{}, err
	}
	defer unlock()

	journal, err := s.journal.Read()
	if err != nil {
		return Recovery{}, err
	}

	if journal.Operation == nil || journal.Operation.ID != operationID {
		return Recovery{}, ErrRecoveryEndedDuringUpload
	}

	journal.Operation.ArchiveSha256 = archiveSha256
	if err := s.journal.Write(journal); err != nil {
		return Recovery{}, err
	}

	return newOperationRecovery(*journal.Operation, s.runningRelease), nil
}

// release forgets an upload that failed before it was answered, so it leaves no record: the
// client already got the reason.
func (s *UploadService) release(ctx context.Context, operationID uuid.UUID) {
	err := s.withOperation(operationID, func(journal Journal) error {
		_, err := s.publisher.Discard(context.WithoutCancel(ctx), journal)

		return err
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to release a refused recovery upload",
			slog.String("operation_id", operationID.String()), slog.Any("error", err))
	}
}

func (s *UploadService) prepare(ctx context.Context, operationID uuid.UUID) error {
	s.preparationMutex.Lock()
	defer s.preparationMutex.Unlock()

	journal, err := s.journal.Read()
	if err != nil {
		return err
	}

	if journal.Operation == nil || journal.Operation.ID != operationID || journal.Operation.Phase != PhasePreparing {
		return nil
	}

	preparationCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.cancellations.Register(operationID, cancel)
	defer s.cancellations.Unregister(operationID)

	if _, isCancelRequested := s.cancelRequests.Load(operationID); isCancelRequested {
		cancel()
	}

	operation := *journal.Operation
	prepareErr := s.prepareStaged(preparationCtx, &operation)

	if preparationCtx.Err() != nil {
		// A cancellation ends the operation itself, and a stopping server leaves it to the next
		// start, which records it as interrupted.
		return nil
	}

	if prepareErr != nil {
		return s.fail(ctx, operationID, prepareErr)
	}

	return s.withOperation(operationID, func(journal Journal) error {
		expiresAt := s.clock.Now().Add(preparedLifetime)
		operation.Phase = PhaseImported
		operation.ExpiresAt = &expiresAt
		journal.Operation = &operation

		s.logger.InfoContext(ctx, "recovery upload prepared",
			slog.String("operation_id", operationID.String()),
			slog.String("source_release", operation.SourceRelease))

		return s.journal.Write(journal)
	})
}

func (s *UploadService) prepareStaged(ctx context.Context, operation *Operation) error {
	keyContent, err := os.ReadFile(s.directory.getStagedKeyPath(operation.ID))
	if err != nil {
		return fmt.Errorf("read the staged master key: %w", err)
	}

	masterKey, err := crypto.NewMasterKey(keyContent)
	if err != nil {
		return newInvalidKeyRefusal(err)
	}

	return s.preparer.Prepare(ctx, operation, s.directory.getStagedArchivePath(operation.ID), masterKey)
}

func (s *UploadService) fail(ctx context.Context, operationID uuid.UUID, prepareErr error) error {
	failure := getFailure(prepareErr)

	s.logger.WarnContext(ctx, "recovery upload could not be prepared; the current installation is unchanged",
		slog.String("operation_id", operationID.String()),
		slog.String("failure", string(failure)),
		slog.Any("error", prepareErr))

	return s.withOperation(operationID, func(journal Journal) error {
		_, err := s.publisher.End(context.WithoutCancel(ctx), journal, &failure)

		return err
	})
}

func (s *UploadService) expire(ctx context.Context, operationID uuid.UUID) error {
	s.logger.InfoContext(ctx, "prepared recovery expired without apply; the current installation is unchanged",
		slog.String("operation_id", operationID.String()))

	return s.withOperation(operationID, func(journal Journal) error {
		if journal.IsPublicationUnfinished() || journal.Operation.Phase != PhaseImported {
			return nil
		}

		_, err := s.publisher.End(context.WithoutCancel(ctx), journal, new(FailureExpired))

		return err
	})
}

// withOperation runs change under the journal lock while the operation is still the journal's,
// and does nothing once it ended.
func (s *UploadService) withOperation(operationID uuid.UUID, change func(journal Journal) error) error {
	unlock, err := s.journal.Lock()
	if err != nil {
		return err
	}
	defer unlock()

	journal, err := s.journal.Read()
	if err != nil {
		return err
	}

	if journal.Operation == nil || journal.Operation.ID != operationID {
		return nil
	}

	return change(journal)
}

// getEndedForCancel repeats the answer of a cancellation that already happened.
func (s *UploadService) getEndedForCancel(journal Journal, recoveryID uuid.UUID) (Recovery, error) {
	if journal.LastEnded != nil && journal.LastEnded.OperationID == recoveryID {
		endedRecovery := newEndedRecovery(*journal.LastEnded, s.runningRelease)
		if endedRecovery.Status == StatusCancelled {
			return endedRecovery, nil
		}

		return Recovery{}, ErrRecoveryNotCancellable
	}

	if journal.LastCompleted != nil && journal.LastCompleted.OperationID == recoveryID {
		return Recovery{}, ErrRecoveryNotCancellable
	}

	return Recovery{}, ErrRecoveryNotFound
}

func (s *UploadService) wake() {
	select {
	case s.wakeSignal <- struct{}{}:
	default:
	}
}

// getFailure turns a preparation error into the safe reason root admin sees.
func getFailure(err error) Failure {
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		return FailureImportFailed
	}

	switch refusal.Reason {
	case RefusalInvalidMasterKey:
		return FailureMasterKeyInvalid
	case RefusalInvalidArchive:
		if errors.Is(err, archive.ErrArchiveIncomplete) {
			return FailureArchiveIncomplete
		}

		return FailureArchiveInvalid
	case RefusalIncompatibleRelease:
		return FailureReleaseMismatch
	case RefusalIncompatibleSchema:
		return FailureSchemaIncompatible
	case RefusalInsufficientDisk:
		return FailureInsufficientDiskSpace
	case RefusalExistingAccounts, RefusalChangedSettings, RefusalUserData, RefusalUnknownState,
		RefusalNewerSchema, RefusalInputUnreadable, RefusalNoExclusiveAccess, RefusalImportFailed:
		return FailureImportFailed
	default:
		return FailureImportFailed
	}
}
