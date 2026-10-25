package stream_guard

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/cache"
)

const (
	downloadLockPrefix = "backup_download_lock:"
	// downloadLockTTL must exceed downloadHeartbeatDelay so an in-flight stream's
	// heartbeat keeps renewing the lock, while a stream that dies without
	// releasing it self-heals within the TTL instead of locking the user out
	// until the cache's default expiry (10 min).
	downloadLockTTL        = 5 * time.Second
	downloadLockValue      = "1"
	downloadHeartbeatDelay = 3 * time.Second
)

type Tracker struct {
	locks *cache.JSONStore[string]
}

func NewTracker(store cache.Store) *Tracker {
	return &Tracker{
		locks: cache.NewJSONStore[string](store, downloadLockPrefix),
	}
}

func (t *Tracker) AcquireDownloadLock(ctx context.Context, userID uuid.UUID) error {
	isCreated, err := t.locks.CreateIfAbsent(
		ctx,
		cache.ExpiringValue[string]{
			Key:      userID.String(),
			Value:    downloadLockValue,
			Lifetime: downloadLockTTL,
		},
	)
	if err != nil {
		return err
	}
	if !isCreated {
		return ErrDownloadAlreadyInProgress
	}

	return nil
}

func (t *Tracker) RefreshDownloadLock(ctx context.Context, userID uuid.UUID) error {
	return t.locks.SetWithLifetime(ctx, cache.ExpiringValue[string]{
		Key:      userID.String(),
		Value:    downloadLockValue,
		Lifetime: downloadLockTTL,
	})
}

func (t *Tracker) ReleaseDownloadLock(ctx context.Context, userID uuid.UUID) error {
	return t.locks.Delete(ctx, userID.String())
}

func (t *Tracker) IsDownloadInProgress(ctx context.Context, userID uuid.UUID) (bool, error) {
	existingLock, err := t.locks.Get(ctx, userID.String())
	if err != nil {
		return false, err
	}

	return existingLock != nil, nil
}

func GetDownloadHeartbeatInterval() time.Duration {
	return downloadHeartbeatDelay
}
