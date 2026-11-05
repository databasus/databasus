// Package download issues the single-use tokens of backup file downloads, streams the files and
// keeps the user streams: at most one heavy stream per user, a download or the restore set of a
// physical backup.
package download

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/cache"
)

const (
	userStreamNamespace = "backup_download_lock:"
	// userStreamLifetime exceeds UserStreamHeartbeatInterval, so the heartbeat of a running stream
	// keeps it, while a stream that dies without releasing it frees the user within the lifetime.
	userStreamLifetime = 5 * time.Second
	userStreamValue    = "1"
	// UserStreamHeartbeatInterval is how often a running stream refreshes its user stream.
	UserStreamHeartbeatInterval = 3 * time.Second
)

// UserStreams holds the user streams in the cache, so they expire on their own.
type UserStreams struct {
	locks *cache.JSONStore[string]
}

func NewUserStreams(store cache.Store) *UserStreams {
	return &UserStreams{locks: cache.NewJSONStore[string](store, userStreamNamespace)}
}

// UserStream is the one heavy stream a user runs. Its holder refreshes it every
// UserStreamHeartbeatInterval and releases it when the stream ends.
type UserStream struct {
	userID  uuid.UUID
	streams *UserStreams
}

// Acquire answers ErrUserStreamHeld while another stream of the user runs.
func (s *UserStreams) Acquire(ctx context.Context, userID uuid.UUID) (UserStream, error) {
	isCreated, err := s.locks.CreateIfAbsent(ctx, cache.ExpiringValue[string]{
		Key:      userID.String(),
		Value:    userStreamValue,
		Lifetime: userStreamLifetime,
	})
	if err != nil {
		return UserStream{}, err
	}

	if !isCreated {
		return UserStream{}, ErrUserStreamHeld
	}

	return UserStream{userID: userID, streams: s}, nil
}

// IsHeld fails closed: a cache that cannot answer reports the stream as held.
func (s *UserStreams) IsHeld(ctx context.Context, userID uuid.UUID) (bool, error) {
	existingLock, err := s.locks.Get(ctx, userID.String())
	if err != nil {
		return true, err
	}

	return existingLock != nil, nil
}

func (s UserStream) Refresh(ctx context.Context) error {
	return s.streams.locks.SetWithLifetime(ctx, cache.ExpiringValue[string]{
		Key:      s.userID.String(),
		Value:    userStreamValue,
		Lifetime: userStreamLifetime,
	})
}

func (s UserStream) Release(ctx context.Context) error {
	return s.streams.locks.Delete(ctx, s.userID.String())
}
