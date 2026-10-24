package pubsub

import (
	"sync"

	"databasus-backend/internal/util/logger"
)

var getDefaultBroker = sync.OnceValue(func() Broker {
	return NewLocalBroker(logger.GetLogger())
})

func GetBroker() Broker {
	return getDefaultBroker()
}
