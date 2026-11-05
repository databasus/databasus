package restorestream

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/cache"
)

const restoreTokenPrefix = "physical_restore_token:"

// TokenLifetime is longer than a download token's 5 min: a physical restore
// stream can be large (full + incrementals + WAL) and is often piped straight
// into an extract, so the window to start it must comfortably outlast a human
// pasting the curl command.
const TokenLifetime = 15 * time.Minute

// Token authorizes one agent-less physical restore stream. Unlike a download
// token (one backup file) it is keyed by a restore SPEC — a database and an
// optional point-in-time — because the stream is resolved from many artifacts
// at request time, not a single stored object.
//
// It lives only in the cache, never in PostgreSQL: the secret token string is
// the key, the cache TTL expires it, and GETDEL consumes it exactly once. So it
// carries no persisted ID, expiry, or used flag — the store provides all three.
type Token struct {
	DatabaseID uuid.UUID  `json:"databaseId"`
	UserID     uuid.UUID  `json:"userId"`
	TargetTime *time.Time `json:"targetTime"`

	// BackupID, when set, switches the stream to a per-backup restore (the FULL
	// plus its incremental ancestors, no WAL) instead of the point-in-time path
	// driven by TargetTime. Exactly one of BackupID / TargetTime semantics
	// applies: a non-nil BackupID takes precedence and TargetTime is ignored.
	BackupID *uuid.UUID `json:"backupId"`
}

type store struct {
	tokens *cache.JSONStore[Token]
}

func newStore(cacheStore cache.Store) *store {
	return &store{
		tokens: cache.NewJSONStore[Token](cacheStore, restoreTokenPrefix),
	}
}

func (s *store) issue(ctx context.Context, token string, restoreToken Token) error {
	return s.tokens.SetWithLifetime(ctx, cache.ExpiringValue[Token]{
		Key:      token,
		Value:    restoreToken,
		Lifetime: TokenLifetime,
	})
}

// consume atomically reads and deletes the token, returning nil when it is
// missing, expired, or already consumed.
func (s *store) consume(ctx context.Context, token string) (*Token, error) {
	return s.tokens.ReadAndDelete(ctx, token)
}
