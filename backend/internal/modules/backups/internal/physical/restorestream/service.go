package restorestream

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/modules/backups/internal/download"
	"databasus-backend/internal/modules/storages"
)

// Service issues and consumes single-use tokens authorizing a physical restore stream. The stream
// takes the user stream of the backups module, so a user cannot run a download and a restore stream
// at once.
type Service struct {
	store   *store
	streams *download.UserStreams
	*Writer
	logger *slog.Logger
	clock  clock.Source
}

type Deps struct {
	Cache    cache.Store
	Streams  *download.UserStreams
	Storages storages.Module
	Logger   *slog.Logger
	Clock    clock.Source
}

func New(deps Deps) *Service {
	return &Service{
		store:   newStore(deps.Cache),
		streams: deps.Streams,
		Writer:  &Writer{storages: deps.Storages},
		logger:  deps.Logger,
		clock:   deps.Clock,
	}
}

type IssuedToken struct {
	Token     string
	ExpiresAt time.Time
}

func (s *Service) IssueToken(ctx context.Context, subject Token) (IssuedToken, error) {
	if err := s.refuseWhileStreamHeld(ctx, subject.UserID); err != nil {
		return IssuedToken{}, err
	}

	token := generateSecureToken()
	expiresAt := s.clock.Now().Add(TokenLifetime)
	if err := s.store.issue(ctx, token, subject); err != nil {
		return IssuedToken{}, err
	}

	s.logger.InfoContext(
		ctx,
		"generated restore token",
		slog.Any("database_id", subject.DatabaseID),
		slog.Any("user_id", subject.UserID),
	)

	return IssuedToken{Token: token, ExpiresAt: expiresAt}, nil
}

// ValidateAndConsumeRestoreToken consumes a restore token, then acquires the user stream. The caller
// must release the stream when the restore stream ends.
//
// The token is consumed (GETDEL) BEFORE the stream is acquired, so it can be
// validated at most once even under concurrency. The trade-off: if the stream
// cannot be acquired (a competing stream slipped in after the token was
// issued), the token is already spent and the user must request a new one. That
// race is narrow — IssueToken already refuses while a stream is held —
// and burning the token beats a validate-then-consume window that two requests
// could both pass.
func (s *Service) ValidateAndConsumeRestoreToken(
	ctx context.Context,
	token string,
) (*Token, download.UserStream, error) {
	restoreToken, err := s.store.consume(ctx, token)
	if err != nil {
		return nil, download.UserStream{}, err
	}
	if restoreToken == nil {
		return nil, download.UserStream{}, ErrInvalidToken
	}

	userStream, err := s.streams.Acquire(ctx, restoreToken.UserID)
	if err != nil {
		return nil, download.UserStream{}, err
	}

	s.logger.InfoContext(
		ctx,
		"restore token validated and consumed",
		slog.Any("database_id", restoreToken.DatabaseID),
		slog.Any("user_id", restoreToken.UserID),
	)

	return restoreToken, userStream, nil
}

func (s *Service) refuseWhileStreamHeld(ctx context.Context, userID uuid.UUID) error {
	isStreamHeld, err := s.streams.IsHeld(ctx, userID)
	if err != nil {
		return err
	}

	if isStreamHeld {
		return download.ErrUserStreamHeld
	}

	return nil
}

// generateSecureToken returns a URL-safe 256-bit random token. A read failure from the OS CSPRNG is
// unrecoverable, so it panics rather than hand back a weak token.
func generateSecureToken() string {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		panic("failed to generate secure random token: " + err.Error())
	}

	return base64.URLEncoding.EncodeToString(randomBytes)
}
