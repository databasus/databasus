package walstreaming

import (
	"context"
	"errors"
	"log/slog"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
)

func (s *Streamer) getExpectedSystemIdentifier() string {
	if s.spec.SourceDB.SystemIdentifier == nil {
		return ""
	}
	return *s.spec.SourceDB.SystemIdentifier
}

func (s *Streamer) openSourceInspection(ctx context.Context) (*pgphysical.Inspection, error) {
	inspection, err := pgphysical.NewSource(s.spec.SourceDB.Engine).
		OpenVerifiedInspection(ctx, s.getExpectedSystemIdentifier())
	s.stopForSourceIdentity(err)
	return inspection, err
}

func (s *Streamer) stopForSourceIdentity(err error) {
	_, isMismatch := errors.AsType[*pgphysical.SourceIdentityError](err)
	if !isMismatch && !errors.Is(err, pgphysical.ErrSourceIdentityMissing) {
		return
	}
	s.sourceIdentityMu.Lock()
	defer s.sourceIdentityMu.Unlock()
	if s.sourceIdentityErr == nil {
		s.sourceIdentityErr = err
	}
	if s.cancelForSourceIdentity != nil {
		s.cancelForSourceIdentity()
	}
}

func (s *Streamer) recordVerifiedReceiverStarted(ctx context.Context) {
	s.sourceIdentityMu.Lock()
	defer s.sourceIdentityMu.Unlock()
	if s.sourceIdentityErr != nil || ctx.Err() != nil {
		return
	}
	if s.spec.WalStreamerRepo != nil {
		if err := s.spec.WalStreamerRepo.MarkVerifiedStreaming(ctx, s.spec.ClaimID); err != nil {
			s.logger.ErrorContext(ctx, "failed to record verified wal streaming", slog.Any("error", err))
			return
		}
	}
	s.receiverRunStartedAtUnixNano.Store(s.spec.Clock.Now().UnixNano())
}
