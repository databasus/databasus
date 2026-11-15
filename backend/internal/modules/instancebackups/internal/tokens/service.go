package tokens

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/clock"
)

type Deps struct {
	Repositories Repositories
	Clock        clock.Source
}

type Service struct {
	tokens *tokenRepository
	clock  clock.Source
}

func NewService(deps Deps) *Service {
	return &Service{tokens: deps.Repositories.tokens, clock: deps.Clock}
}

// Issuance names who gets a token, for what and for how long.
type Issuance struct {
	Purpose  Purpose
	UserID   uuid.UUID
	CopyID   *uuid.UUID
	Lifetime time.Duration
}

func (s *Service) Issue(ctx context.Context, issuance Issuance) (IssuedToken, error) {
	tokenText := generateTokenText()
	expiresAt := s.clock.Now().Add(issuance.Lifetime)

	err := s.tokens.insert(ctx, Token{
		TokenSha256: hashToken(tokenText),
		Purpose:     issuance.Purpose,
		UserID:      issuance.UserID,
		CopyID:      issuance.CopyID,
		ExpiresAt:   expiresAt,
		ConsumedAt:  nil,
	})
	if err != nil {
		return IssuedToken{}, err
	}

	return IssuedToken{Token: tokenText, ExpiresAt: expiresAt}, nil
}

// Consume spends the token once; it reports false for anything that cannot be spent.
func (s *Service) Consume(ctx context.Context, tokenText string, purpose Purpose) (Token, bool, error) {
	return s.tokens.consume(ctx, hashToken(tokenText), purpose, s.clock.Now())
}

func (s *Service) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return s.tokens.deleteExpiredBefore(ctx, now)
}

func (s *Service) RevokeAll(ctx context.Context) error {
	return s.tokens.deleteAll(ctx)
}
