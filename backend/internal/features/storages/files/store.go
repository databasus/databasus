package storage_files

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
)

// The only way backup code writes a file to a storage and the only way it hands
// that file over to a catalog transaction. It owns the obligation to remove the
// file until such a transaction claims it.
type Store struct {
	repository     *PendingDeletionRepository
	locator        StorageLocator
	fieldEncryptor encryption.FieldEncryptor
	logger         *slog.Logger
	timings        Timings

	liveWritesMutex sync.Mutex
	liveWrites      map[uuid.UUID]struct{}
}

func NewStore(dependencies Dependencies) *Store {
	return &Store{
		repository:     dependencies.Repository,
		locator:        dependencies.Locator,
		fieldEncryptor: dependencies.FieldEncryptor,
		logger:         dependencies.Logger,
		timings:        dependencies.Timings,
		liveWrites:     make(map[uuid.UUID]struct{}),
	}
}

// The obligation to remove the file exists before the first byte leaves, so a
// provider error, a crash or a caller that never publishes all end the same way,
// with the file removed.
func (s *Store) WriteFile(
	ctx context.Context,
	reference StoredFileReference,
	file io.Reader,
) (WriteReceipt, error) {
	pendingID := uuid.New()

	// Joining before the insert is what makes the worker's exclusion sound: a row
	// it can see belongs to a writer that is already in the set.
	s.joinLiveWrites(pendingID)
	defer s.leaveLiveWrites(pendingID)

	// The obligation commits on its own even when the caller writes from inside a transaction, because
	// a rollback of that transaction must not take the obligation for an uploaded file with it.
	pool := database.Conn(database.WithoutTransaction(ctx), db.GetDb())

	pending, err := s.repository.InsertIfAbsent(pool, pendingID, reference, s.timings.CommitWindow)
	if err != nil {
		return WriteReceipt{}, err
	}

	writer, err := s.locator.GetFileWriter(ctx, reference.StorageID)
	if err != nil {
		s.makeEligibleNow(ctx, pending)

		return WriteReceipt{}, fmt.Errorf("locate storage for write: %w", err)
	}

	logger := s.logger.With("storage_id", reference.StorageID, "file_name", reference.FileName)

	if saveErr := writer.SaveFile(ctx, s.fieldEncryptor, logger, reference.FileName, file); saveErr != nil {
		s.makeEligibleNow(ctx, pending)

		return WriteReceipt{}, saveErr
	}

	kept, err := s.repository.SetNotBeforeIfGeneration(
		pool, pending.ID, pending.Generation, s.timings.CommitWindow,
	)
	if err != nil {
		return WriteReceipt{}, err
	}

	if !kept {
		return WriteReceipt{}, ErrFileTakenForDeletion
	}

	return WriteReceipt{
		PendingDeletionID: pending.ID,
		Reference:         reference,
		Generation:        pending.Generation,
	}, nil
}

// Fails unless every receipt still matches its row, so a file that cleanup already
// took cannot be published by the transaction that rolls back here. It runs in the
// transaction the context carries.
func (s *Store) ConfirmFileWrites(ctx context.Context, receipts []WriteReceipt) error {
	tx := database.Conn(ctx, db.GetDb())
	if err := requireTransaction(tx); err != nil {
		return err
	}

	for _, receipt := range receipts {
		released, err := s.repository.CompleteIfGeneration(
			tx, receipt.PendingDeletionID, receipt.Generation,
		)
		if err != nil {
			return err
		}

		if !released {
			return fmt.Errorf("%w: %s", ErrReceiptNoLongerValid, receipt.Reference.FileName)
		}
	}

	return nil
}

// Never fails because a writer is still running or because the same file was
// requested before: a caller removing catalog rows has no way to wait for an
// upload it does not own. It runs in the transaction the context carries.
func (s *Store) RequestFileDeletions(ctx context.Context, references []StoredFileReference) error {
	// A reference with no name cannot become an obligation, and a caller deriving
	// names from a row should not have to know which of them the row carries.
	named := make([]StoredFileReference, 0, len(references))

	for _, reference := range references {
		if reference.FileName != "" {
			named = append(named, reference)
		}
	}

	references = named

	tx := database.Conn(ctx, db.GetDb())
	if err := requireTransaction(tx); err != nil {
		return err
	}

	return s.repository.InsertOrTake(tx, references)
}

func (s *Store) GetTimings() Timings {
	return s.timings
}

// A false result means something already took the file, which leaves it eligible
// anyway. The write is over either way, so the failure is logged, not returned.
func (s *Store) makeEligibleNow(ctx context.Context, pending *PendingDeletion) {
	if _, err := s.repository.SetNotBeforeIfGeneration(
		db.GetDb(), pending.ID, pending.Generation, 0,
	); err != nil {
		s.logger.ErrorContext(ctx, "failed to schedule cleanup for a failed write",
			"storage_id", pending.StorageID,
			"file_name", pending.FileName,
			"error", err)
	}
}

func (s *Store) joinLiveWrites(id uuid.UUID) {
	s.liveWritesMutex.Lock()
	defer s.liveWritesMutex.Unlock()

	s.liveWrites[id] = struct{}{}
}

func (s *Store) leaveLiveWrites(id uuid.UUID) {
	s.liveWritesMutex.Lock()
	defer s.liveWritesMutex.Unlock()

	delete(s.liveWrites, id)
}

// The storage being deleted holds its row lock, so no new write to it can register
// after this snapshot.
func (s *Store) getLiveWriteIDs() map[uuid.UUID]struct{} {
	liveWriteIDs := map[uuid.UUID]struct{}{}

	_ = s.withLiveWrites(func(excluded []uuid.UUID) error {
		for _, id := range excluded {
			liveWriteIDs[id] = struct{}{}
		}

		return nil
	})

	return liveWriteIDs
}

// Holding the mutex across claim is what keeps the exclusion list from going stale
// between reading it and running the statement.
func (s *Store) withLiveWrites(claim func(excluded []uuid.UUID) error) error {
	s.liveWritesMutex.Lock()
	defer s.liveWritesMutex.Unlock()

	excluded := make([]uuid.UUID, 0, len(s.liveWrites))
	for id := range s.liveWrites {
		excluded = append(excluded, id)
	}

	return claim(excluded)
}

// Both batch methods are all-or-nothing. On the shared handle a batch that fails
// partway has already released the receipts it got through, stranding those files
// forever.
func requireTransaction(tx *gorm.DB) error {
	if _, inTransaction := tx.Statement.ConnPool.(gorm.TxCommitter); !inTransaction {
		return ErrTransactionRequired
	}

	return nil
}
