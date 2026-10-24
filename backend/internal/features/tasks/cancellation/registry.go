package task_cancellation

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Registry struct {
	mutex       sync.Mutex
	cancelTasks map[uuid.UUID]context.CancelFunc
	// Closed while no task is registered; replaced by an open channel when the first task registers.
	noTasksRegistered chan struct{}
}

func NewRegistry() *Registry {
	noTasksRegistered := make(chan struct{})
	close(noTasksRegistered)

	return &Registry{
		cancelTasks:       make(map[uuid.UUID]context.CancelFunc),
		noTasksRegistered: noTasksRegistered,
	}
}

func (r *Registry) RegisterTask(taskID uuid.UUID, cancelTask context.CancelFunc) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.cancelTasks) == 0 {
		r.noTasksRegistered = make(chan struct{})
	}

	r.cancelTasks[taskID] = cancelTask
}

func (r *Registry) UnregisterTask(taskID uuid.UUID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.removeTask(taskID)
}

func (r *Registry) CancelRegisteredTask(taskID uuid.UUID) {
	r.mutex.Lock()
	cancelTask, isRegistered := r.cancelTasks[taskID]
	if isRegistered {
		r.removeTask(taskID)
	}
	r.mutex.Unlock()

	if !isRegistered {
		return
	}

	cancelTask()
}

func (r *Registry) WaitUntilNoTasksRegistered(ctx context.Context) error {
	r.mutex.Lock()
	noTasksRegistered := r.noTasksRegistered
	r.mutex.Unlock()

	select {
	case <-noTasksRegistered:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Registry) removeTask(taskID uuid.UUID) {
	if _, isRegistered := r.cancelTasks[taskID]; !isRegistered {
		return
	}

	delete(r.cancelTasks, taskID)

	if len(r.cancelTasks) == 0 {
		close(r.noTasksRegistered)
	}
}
