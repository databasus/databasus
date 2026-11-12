package iox

import (
	"context"
	"fmt"
	"log/slog"
)

type RequestRetrySpec struct {
	Logger          *slog.Logger
	RetryPolicySpec TransferRetryPolicySpec

	// RequestDescription names the request in the error returned once the budget is spent, so a
	// failed save says which request failed, such as "part 368 of backups/x.part000005".
	RequestDescription string
	LogAttributes      []slog.Attr

	IsRetryableError func(error) bool
	Send             func(ctx context.Context) error
}

// RetryRequest repeats a request that a storage refused with a temporary error, on top of the
// short retries its client library makes. Every call gets a fresh budget, so refusals spread
// across a long upload never add up. The request must be safe to send twice: Send is called again
// with the same content.
func RetryRequest(ctx context.Context, spec RequestRetrySpec) error {
	retryPolicy := NewTransferRetryPolicy(spec.RetryPolicySpec)

	for {
		err := spec.Send(ctx)
		if err == nil {
			return nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%s cancelled: %w", spec.RequestDescription, ctxErr)
		}

		if !spec.IsRetryableError(err) {
			return err
		}

		retryPolicy.RegisterFailedAttempt(false)

		if retryPolicy.IsExhausted() {
			return fmt.Errorf(
				"%w: %s failed after %d attempts: %w",
				ErrRetryBudgetExhausted, spec.RequestDescription, retryPolicy.GetAttemptsSinceProgress(), err,
			)
		}

		logRefusedRequest(ctx, spec, retryPolicy, err)

		if err := retryPolicy.WaitBeforeRetry(ctx); err != nil {
			return fmt.Errorf("%s cancelled: %w", spec.RequestDescription, err)
		}
	}
}

func logRefusedRequest(ctx context.Context, spec RequestRetrySpec, retryPolicy *TransferRetryPolicy, cause error) {
	attributes := append([]slog.Attr{}, spec.LogAttributes...)
	attributes = append(attributes,
		slog.Int("attempt", retryPolicy.GetAttemptsSinceProgress()),
		slog.Int("max_attempts", retryPolicy.GetMaxAttemptsWithoutProgress()),
		slog.Any("error", cause),
	)

	spec.Logger.LogAttrs(ctx, slog.LevelWarn, "storage request refused, retrying", attributes...)
}
