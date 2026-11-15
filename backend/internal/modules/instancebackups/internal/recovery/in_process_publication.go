package recovery

import (
	"context"
	"log/slog"
)

// PublishAccepted publishes, in a running server, the recovery whose apply root admin accepted. It
// runs after the application stopped its requests and work and closed its connections, and it
// leaves the lock shared again for the application built next.
//
// When the work did not stop in time, or another process uses the data directory, publication
// would replace a database still in use, so the recovery ends as QUIESCENCE_FAILED and the active
// database and key stay as they are.
func (s *Startup) PublishAccepted(ctx context.Context, lock *LifecycleLock, isDrained bool) error {
	journal, err := s.journal.Read()
	if err != nil {
		return err
	}

	if journal.Operation == nil || !journal.IsPublicationUnfinished() {
		return nil
	}

	if !isDrained {
		return s.endUnquiesced(ctx, journal, "the running work did not stop within the drain limit")
	}

	isExclusive, err := lock.TryExclusive()
	if err != nil {
		return err
	}

	if !isExclusive {
		if err := s.shareAgain(lock); err != nil {
			return err
		}

		return s.endUnquiesced(ctx, journal, "another Databasus server or console command uses this data directory")
	}

	s.logger.InfoContext(ctx, "publishing the accepted recovery",
		slog.String("operation_id", journal.Operation.ID.String()))

	journal, err = s.publisher.Publish(ctx, journal)
	if err != nil {
		return err
	}

	if err := s.checkPublishedKey(journal); err != nil {
		return err
	}

	return s.shareAgain(lock)
}

func (s *Startup) endUnquiesced(ctx context.Context, journal Journal, explanation string) error {
	s.logger.WarnContext(ctx, "the accepted recovery was not published",
		slog.String("operation_id", journal.Operation.ID.String()),
		slog.String("explanation", explanation),
		slog.String("current_installation", "unchanged"),
		slog.String("next_step", "stop other Databasus processes and upload the Databasus backup file again"))

	_, err := s.publisher.End(ctx, journal, new(FailureQuiescenceFailed))

	return err
}

// shareAgain returns the lock to shared. A failed conversion to exclusive can drop the shared hold,
// so it is taken again.
func (s *Startup) shareAgain(lock *LifecycleLock) error {
	isShared, err := lock.TryShared()
	if err != nil {
		return err
	}

	if !isShared {
		return ErrLifecycleBusy
	}

	return nil
}
