package restores_core

import "context"

// NewRestoreContext derives the context a restore runs under. Besides ending with executionCtx,
// it ends with ErrRestoreShutdown once shutdownSignal closes, so the restorer can tell a stop of
// the application from a user's cancellation.
func NewRestoreContext(
	executionCtx context.Context,
	shutdownSignal <-chan struct{},
) (context.Context, context.CancelCauseFunc) {
	restoreCtx, cancel := context.WithCancelCause(executionCtx)

	go func() {
		select {
		case <-restoreCtx.Done():
		case <-shutdownSignal:
			cancel(ErrRestoreShutdown)
		}
	}()

	return restoreCtx, cancel
}
