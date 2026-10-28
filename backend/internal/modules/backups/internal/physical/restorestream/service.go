package restorestream

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/cache"
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
}

type Deps struct {
	Cache    cache.Store
	Streams  *download.UserStreams
	Storages storages.Module
	Logger   *slog.Logger
}

func New(deps Deps) *Service {
	return &Service{
		store:   newStore(deps.Cache),
		streams: deps.Streams,
		Writer:  &Writer{storages: deps.Storages},
		logger:  deps.Logger,
	}
}

// GenerateRestoreToken issues a single-use, short-TTL token authorizing a
// physical restore stream of databaseID to targetTime (nil ⇒ latest).
func (s *Service) GenerateRestoreToken(
	ctx context.Context,
	databaseID, userID uuid.UUID,
	targetTime *time.Time,
) (string, error) {
	err := s.refuseWhileStreamHeld(ctx, userID)
	if err != nil {
		return "", err
	}

	token := generateSecureToken()

	if err := s.store.issue(ctx, token, Token{
		DatabaseID: databaseID,
		UserID:     userID,
		TargetTime: targetTime,
	}); err != nil {
		return "", err
	}

	s.logger.InfoContext(
		ctx,
		"generated restore token",
		slog.Any("database_id", databaseID),
		slog.Any("user_id", userID),
	)

	return token, nil
}

// GenerateBackupRestoreToken issues a single-use, short-TTL token authorizing a
// per-backup restore stream (the FULL plus its incremental ancestors, no WAL)
// of backupID within databaseID.
func (s *Service) GenerateBackupRestoreToken(
	ctx context.Context,
	databaseID, userID, backupID uuid.UUID,
) (string, error) {
	err := s.refuseWhileStreamHeld(ctx, userID)
	if err != nil {
		return "", err
	}

	token := generateSecureToken()

	if err := s.store.issue(ctx, token, Token{
		DatabaseID: databaseID,
		UserID:     userID,
		BackupID:   &backupID,
	}); err != nil {
		return "", err
	}

	s.logger.InfoContext(
		ctx,
		"generated backup restore token",
		slog.Any("database_id", databaseID),
		slog.Any("user_id", userID),
		slog.Any("backup_id", backupID),
	)

	return token, nil
}

// ValidateAndConsumeRestoreToken consumes a restore token, then acquires the user stream. The caller
// must release the stream when the restore stream ends.
//
// The token is consumed (GETDEL) BEFORE the stream is acquired, so it can be
// validated at most once even under concurrency. The trade-off: if the stream
// cannot be acquired (a competing stream slipped in after the token was
// issued), the token is already spent and the user must request a new one. That
// race is narrow — GenerateRestoreToken already refuses while a stream is held —
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
