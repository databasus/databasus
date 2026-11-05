package module

import (
	"context"
	"log/slog"
	"time"

	"databasus-backend/internal/lib/background"
)

const (
	expiredCodesCleanupJobName  = "users_expired_codes_cleanup"
	expiredCodesCleanupInterval = 10 * time.Minute
)

// newExpiredCodesCleanupJob deletes the single-use codes nobody comes back for: the pending sign-ins
// and the reset codes the hourly caps no longer count.
func (m *Module) newExpiredCodesCleanupJob() background.Job {
	return background.Job{
		Name:     expiredCodesCleanupJobName,
		Interval: expiredCodesCleanupInterval,
		Run: func(ctx context.Context, now time.Time) error {
			deletedPendingSignInCount, err := m.signIn.DeletePendingSignInsPastRetention(ctx, now)
			if err != nil {
				return err
			}

			deletedResetCodeCount, err := m.passwordReset.DeleteResetCodesPastRetention(ctx, now)
			if err != nil {
				return err
			}

			m.logger.InfoContext(ctx, "expired codes deleted",
				slog.Int64("deleted_pending_sign_in_count", deletedPendingSignInCount),
				slog.Int64("deleted_reset_code_count", deletedResetCodeCount))

			return nil
		},
	}
}
