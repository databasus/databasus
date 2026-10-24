package task_cancellation

import (
	"context"
	"sync"

	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/pubsub"
)

var (
	getTaskCancellationRegistry  = sync.OnceValue(NewRegistry)
	getTaskCancellationRequester = sync.OnceValue(func() *Requester {
		return NewRequester(pubsub.GetBroker(), logger.GetLogger())
	})
	getTaskCancellationListener = sync.OnceValue(func() *Listener {
		return NewListener(
			pubsub.GetBroker(),
			getTaskCancellationRegistry(),
			logger.GetLogger(),
		)
	})
)

func GetRegistry() *Registry {
	return getTaskCancellationRegistry()
}

func GetRequester() *Requester {
	return getTaskCancellationRequester()
}

func GetListener() *Listener {
	return getTaskCancellationListener()
}

var SetupDependencies = sync.OnceFunc(func() {
	if err := getTaskCancellationListener().Start(context.Background()); err != nil {
		panic(err)
	}
})
