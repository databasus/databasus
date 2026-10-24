package task_cancellation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Registry_CancelRegisteredTask_CancelsAndRemovesTask(t *testing.T) {
	registry := NewRegistry()
	taskID := uuid.New()
	taskContext, cancelTask := context.WithCancel(t.Context())
	registry.RegisterTask(taskID, cancelTask)

	registry.CancelRegisteredTask(taskID)
	assert.ErrorIs(t, taskContext.Err(), context.Canceled)

	secondContext, secondCancelTask := context.WithCancel(t.Context())
	defer secondCancelTask()
	registry.RegisterTask(taskID, secondCancelTask)
	registry.UnregisterTask(taskID)
	registry.CancelRegisteredTask(taskID)
	assert.NoError(t, secondContext.Err())
}

func Test_Registry_WaitUntilNoTasksRegistered_WhenLastTaskUnregisters_Returns(t *testing.T) {
	registry := NewRegistry()
	firstTaskID := uuid.New()
	secondTaskID := uuid.New()
	registry.RegisterTask(firstTaskID, func() {})
	registry.RegisterTask(secondTaskID, func() {})

	waitResult := make(chan error, 1)
	go func() { waitResult <- registry.WaitUntilNoTasksRegistered(t.Context()) }()

	registry.UnregisterTask(firstTaskID)
	registry.CancelRegisteredTask(secondTaskID)

	select {
	case waitErr := <-waitResult:
		assert.NoError(t, waitErr)
	case <-time.After(5 * time.Second):
		t.Fatal("wait did not return after the last task unregistered")
	}
}

func Test_Registry_WaitUntilNoTasksRegistered_WhenNoTaskRegistered_ReturnsAtOnce(t *testing.T) {
	registry := NewRegistry()
	taskID := uuid.New()
	registry.RegisterTask(taskID, func() {})
	registry.UnregisterTask(taskID)

	waitCtx, cancelWait := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelWait()

	assert.NoError(t, registry.WaitUntilNoTasksRegistered(waitCtx))
}

func Test_Registry_WaitUntilNoTasksRegistered_WhenContextEndsWithTaskRegistered_ReturnsContextError(
	t *testing.T,
) {
	registry := NewRegistry()
	registry.RegisterTask(uuid.New(), func() {})

	waitCtx, cancelWait := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancelWait()

	assert.ErrorIs(t, registry.WaitUntilNoTasksRegistered(waitCtx), context.DeadlineExceeded)
}
