package files

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Sharing a checked-out session keeps overlapping uploads from reserving the whole database pool.
type WriteProtectionRepository struct {
	db           *gorm.DB
	mutex        sync.Mutex
	connection   *sql.Conn
	activeWrites map[uuid.UUID]writeProtectionRequest
	failure      error
}

type writeProtectionRequest struct {
	PendingDeletionID uuid.UUID
	Reference         StoredFileReference
	CancelWrite       context.CancelCauseFunc
}

func NewWriteProtectionRepository(db *gorm.DB) *WriteProtectionRepository {
	return &WriteProtectionRepository{
		db: db, mutex: sync.Mutex{}, connection: nil,
		activeWrites: make(map[uuid.UUID]writeProtectionRequest), failure: nil,
	}
}

func (r *WriteProtectionRepository) ProtectWrite(ctx context.Context, request writeProtectionRequest) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	if r.failure != nil {
		return r.failure
	}

	if r.connection == nil {
		pool, err := r.db.DB()
		if err != nil {
			return fmt.Errorf("get write protection pool: %w", err)
		}
		connection, err := pool.Conn(ctx)
		if err != nil {
			return fmt.Errorf("reserve write protection connection: %w", err)
		}
		r.connection = connection
	}

	guardCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	var isProtected bool
	err := r.connection.QueryRowContext(guardCtx,
		"SELECT pg_try_advisory_lock(hashtextextended('storages-file-write:' || $1 || ':' || $2,0))",
		request.Reference.StorageID.String(), request.Reference.FileName).Scan(&isProtected)
	if err != nil {
		r.failProtection(err)
		protectionErr := r.failure
		r.closeIdleConnection()
		return fmt.Errorf("protect storage file write: %w", protectionErr)
	}
	if !isProtected {
		r.closeIdleConnection()
		return ErrFileNameAlreadyRegistered
	}

	r.activeWrites[request.PendingDeletionID] = request

	return nil
}

func (r *WriteProtectionRepository) ReleaseWrite(ctx context.Context, id uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	guardCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	err := r.failure
	if err == nil {
		var isReleased bool
		reference := r.activeWrites[id].Reference
		err = r.connection.QueryRowContext(guardCtx,
			"SELECT pg_advisory_unlock(hashtextextended('storages-file-write:' || $1 || ':' || $2,0))",
			reference.StorageID.String(), reference.FileName).Scan(&isReleased)
		if err == nil && !isReleased {
			err = ErrWriteProtectionLost
		}
	}

	delete(r.activeWrites, id)

	if err != nil {
		r.failProtection(err)
		err = r.failure
	}
	r.closeIdleConnection()

	return err
}

func (r *WriteProtectionRepository) TryLockFileForDeletion(
	ctx context.Context,
	reference StoredFileReference,
) (bool, error) {
	var canDelete bool
	err := database.Conn(ctx, r.db).
		Raw("SELECT pg_try_advisory_xact_lock(hashtextextended('storages-file-write:' || ?::text || ':' || ?,0))", reference.StorageID, reference.FileName).
		Scan(&canDelete).Error
	if err != nil {
		return false, fmt.Errorf("check storage file write protection: %w", err)
	}
	return canDelete, nil
}

func (r *WriteProtectionRepository) failProtection(cause error) {
	r.failure = fmt.Errorf("%w: %w", ErrWriteProtectionLost, cause)
	for _, write := range r.activeWrites {
		write.CancelWrite(r.failure)
	}
	if r.connection != nil {
		_ = r.connection.Raw(func(any) error { return driver.ErrBadConn })
		_ = r.connection.Close()
		r.connection = nil
	}
}

func (r *WriteProtectionRepository) closeIdleConnection() {
	if len(r.activeWrites) != 0 {
		return
	}
	if r.connection != nil {
		_ = r.connection.Close()
		r.connection = nil
	}
	r.failure = nil
}
