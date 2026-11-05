package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
)

// Deps is what both the store and the deletion job need, so the two constructors cannot drift apart.
type Deps struct {
	Repository      *PendingDeletionRepository
	WriteProtection *WriteProtectionRepository
	Transactor      *database.Transactor
	Locator         StorageLocator
	Logger          *slog.Logger
	Clock           clock.Source
	Timings         Timings
	// Uploads runs the upload of each background write.
	Uploads *background.Group
}

// Store is the only way backup code writes a file to a storage and the only way it hands that file
// over to a catalog transaction. It owns the obligation to remove the file until such a transaction
// claims it.
type Store struct {
	repository      *PendingDeletionRepository
	writeProtection *WriteProtectionRepository
	transactor      *database.Transactor
	locator         StorageLocator
	logger          *slog.Logger
	clock           clock.Source
	timings         Timings
	uploads         *background.Group

	liveWritesMutex  sync.Mutex
	liveWrites       map[uuid.UUID]struct{}
	deletionRequests chan struct{}
}

func NewStore(deps Deps) *Store {
	return &Store{
		repository:       deps.Repository,
		writeProtection:  deps.WriteProtection,
		transactor:       deps.Transactor,
		locator:          deps.Locator,
		logger:           deps.Logger,
		clock:            deps.Clock,
		timings:          deps.Timings,
		uploads:          deps.Uploads,
		liveWritesMutex:  sync.Mutex{},
		liveWrites:       make(map[uuid.UUID]struct{}),
		deletionRequests: make(chan struct{}, 1),
	}
}

// WriteFile records the obligation to remove the file before the first byte leaves, so a provider
// error, a crash or a caller that never publishes all end the same way, with the file removed.
func (s *Store) WriteFile(
	ctx context.Context,
	reference StoredFileReference,
	file io.Reader,
) (receipt WriteReceipt, writeErr error) {
	pendingID := uuid.New()
	ctx, cancelWrite := context.WithCancelCause(ctx)
	defer cancelWrite(nil)

	// Joining before the insert is what makes the deletion job's exclusion sound: a row it can see
	// belongs to a writer that is already in the set.
	s.joinLiveWrites(pendingID)
	defer s.leaveLiveWrites(pendingID)

	if err := s.writeProtection.ProtectWrite(ctx, writeProtectionRequest{
		PendingDeletionID: pendingID, Reference: reference, CancelWrite: cancelWrite,
	}); err != nil {
		return WriteReceipt{}, err
	}

	defer func() {
		if err := s.writeProtection.ReleaseWrite(ctx, pendingID); err != nil {
			s.logger.ErrorContext(ctx, "failed to release storage file write protection", slog.Any("error", err))
			receipt = WriteReceipt{}
			writeErr = errors.Join(writeErr, err)
		}
	}()

	// The obligation commits on its own even when the caller writes from inside a transaction, because
	// a rollback of that transaction must not take the obligation for an uploaded file with it.
	poolCtx := database.WithoutTransaction(ctx)

	now := s.clock.Now()

	pending, err := s.repository.InsertIfAbsent(poolCtx, Registration{
		ID:        pendingID,
		Reference: reference,
		NotBefore: now.Add(s.timings.CommitWindow),
		Now:       now,
	})
	if err != nil {
		return WriteReceipt{}, err
	}

	located, err := s.locator.LocateStorage(ctx, reference.StorageID)
	if err != nil {
		s.makeEligibleNow(poolCtx, pending)

		return WriteReceipt{}, fmt.Errorf("locate storage for write: %w", err)
	}

	saveErr := located.Provider.SaveFile(ctx, located.getFileAccess(reference.FileName, s.logger), file)
	if saveErr != nil {
		s.makeEligibleNow(poolCtx, pending)

		return WriteReceipt{}, saveErr
	}

	now = s.clock.Now()

	isKept, err := s.repository.SetNotBeforeIfGeneration(poolCtx, Rescheduling{
		PendingDeletionID: pending.ID,
		Generation:        pending.Generation,
		NotBefore:         now.Add(s.timings.CommitWindow),
		Now:               now,
	})
	if err != nil {
		return WriteReceipt{}, err
	}

	if !isKept {
		return WriteReceipt{}, ErrFileTakenForDeletion
	}

	return WriteReceipt{
		PendingDeletionID: pending.ID,
		Reference:         reference,
		Generation:        pending.Generation,
	}, nil
}

// ConfirmFileWrites fails unless every receipt still matches its row, so a file that cleanup already
// took cannot be published by the transaction that rolls back here. It runs in the transaction ctx
// carries.
func (s *Store) ConfirmFileWrites(ctx context.Context, receipts []WriteReceipt) error {
	if !s.repository.IsInTransaction(ctx) {
		return ErrTransactionRequired
	}

	for _, receipt := range receipts {
		isReleased, err := s.repository.CompleteIfGeneration(ctx, receipt.PendingDeletionID, receipt.Generation)
		if err != nil {
			return err
		}

		if !isReleased {
			return fmt.Errorf("%w: %s", ErrReceiptNoLongerValid, receipt.Reference.FileName)
		}
	}

	return nil
}

// RequestFileDeletions never fails because a writer is still running or because the same file was
// requested before: a caller removing catalog rows has no way to wait for an upload it does not own.
// It runs in the transaction ctx carries.
func (s *Store) RequestFileDeletions(ctx context.Context, references []StoredFileReference) error {
	if !s.repository.IsInTransaction(ctx) {
		return ErrTransactionRequired
	}

	// A reference with no name cannot become an obligation, and a caller deriving names from a row
	// should not have to know which of them the row carries.
	namedReferences := make([]StoredFileReference, 0, len(references))

	for _, reference := range references {
		if reference.FileName != "" {
			namedReferences = append(namedReferences, reference)
		}
	}

	return s.requestFileDeletions(ctx, FileDeletionRequest{References: namedReferences, Scope: nil})
}

// makeEligibleNow logs a failure instead of returning it, because the write is over either way. A
// false result means something already took the file, which leaves it eligible anyway.
func (s *Store) makeEligibleNow(ctx context.Context, pending *PendingDeletion) {
	now := s.clock.Now()

	_, err := s.repository.SetNotBeforeIfGeneration(ctx, Rescheduling{
		PendingDeletionID: pending.ID,
		Generation:        pending.Generation,
		NotBefore:         now,
		Now:               now,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to schedule cleanup for a failed write",
			slog.String("storage_id", pending.StorageID.String()),
			slog.String("file_name", pending.FileName),
			slog.Any("error", err))
	}
}

func (s *Store) joinLiveWrites(id uuid.UUID) {
	s.liveWritesMutex.Lock()
	defer s.liveWritesMutex.Unlock()

	s.liveWrites[id] = struct{}{}
}

func (s *Store) leaveLiveWrites(id uuid.UUID) {
	s.liveWritesMutex.Lock()

	delete(s.liveWrites, id)
	s.liveWritesMutex.Unlock()

	s.requestDeletionPass()
}

// withLiveWrites holds the mutex across claim, which keeps the exclusion list from going stale
// between reading it and running the statement.
func (s *Store) withLiveWrites(claim func(excludedIDs []uuid.UUID) error) error {
	s.liveWritesMutex.Lock()
	defer s.liveWritesMutex.Unlock()

	excludedIDs := make([]uuid.UUID, 0, len(s.liveWrites))
	for id := range s.liveWrites {
		excludedIDs = append(excludedIDs, id)
	}

	return claim(excludedIDs)
}
