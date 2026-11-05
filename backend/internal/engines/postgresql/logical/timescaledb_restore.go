package logical

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
)

// The target survives a failed restore, so cancellation must still allow a cleanup attempt.
func (s *Source) restoreWithTimescaleDBHooks(ctx context.Context, spec RestoreSpec, input io.Reader) (err error) {
	err = s.runTimescaleDBPreRestore(ctx, spec.Logger)
	if err != nil {
		return fmt.Errorf("timescaledb_pre_restore failed: %w", err)
	}

	spec.Logger.InfoContext(ctx, "entered timescaledb restoring mode")

	defer func() {
		newCleanupContext := spec.NewCleanupContext
		if newCleanupContext == nil {
			newCleanupContext = func(ctx context.Context) (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			}
		}
		cleanupCtx, cancelCleanup := newCleanupContext(ctx)
		defer cancelCleanup()

		postErr := cleanupCtx.Err()
		if postErr == nil {
			postErr = s.runTimescaleDBPostRestore(cleanupCtx, spec.Logger)
		}
		if postErr == nil {
			return
		}

		spec.Logger.ErrorContext(
			ctx,
			"timescaledb_post_restore failed; target database may be left in restoring mode",
			slog.Any("error", postErr),
		)

		if err == nil {
			err = fmt.Errorf("timescaledb_post_restore failed: %w", postErr)
		}
	}()

	err = s.restoreArchive(ctx, spec, input)
	if err != nil && ctx.Err() == nil {
		err = &ArchiveRestoreError{Err: err}
	}

	return err
}

func (s *Source) runTimescaleDBPreRestore(ctx context.Context, logger *slog.Logger) error {
	return s.execStatements(ctx, logger, pgrestore.GetTimescaleDBPreRestoreStatements()...)
}

func (s *Source) runTimescaleDBPostRestore(ctx context.Context, logger *slog.Logger) error {
	return s.execStatements(ctx, logger, pgrestore.GetTimescaleDBPostRestoreStatements()...)
}

func (s *Source) execStatements(ctx context.Context, logger *slog.Logger, statements ...string) error {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", s.settings.Database, err)
	}
	defer closeConn(ctx, conn, logger)

	for _, statement := range statements {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to run %q: %w", statement, err)
		}
	}

	return nil
}
