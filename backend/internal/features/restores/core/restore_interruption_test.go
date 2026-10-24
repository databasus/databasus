package restores_core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewRestoreContext_WhenShutdownSignalCloses_EndsWithShutdownCause(t *testing.T) {
	shutdownSignal := make(chan struct{})

	restoreCtx, cancelRestore := NewRestoreContext(t.Context(), shutdownSignal)
	defer cancelRestore(nil)

	close(shutdownSignal)
	<-restoreCtx.Done()

	assert.ErrorIs(t, context.Cause(restoreCtx), ErrRestoreShutdown)
}

func Test_NewRestoreContext_WhenExecutionContextCancelled_EndsWithoutShutdownCause(t *testing.T) {
	executionCtx, cancelExecution := context.WithCancel(t.Context())

	restoreCtx, cancelRestore := NewRestoreContext(executionCtx, make(chan struct{}))
	defer cancelRestore(nil)

	cancelExecution()
	<-restoreCtx.Done()

	assert.ErrorIs(t, context.Cause(restoreCtx), context.Canceled)
	assert.NotErrorIs(t, context.Cause(restoreCtx), ErrRestoreShutdown)
}
