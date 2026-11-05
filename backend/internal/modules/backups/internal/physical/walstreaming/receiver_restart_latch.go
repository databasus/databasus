package walstreaming

import (
	"context"
	"sync"
)

type receiverRestartLatch struct {
	mutex          sync.Mutex
	cancelReceiver context.CancelFunc
	isRequested    bool
}

func (l *receiverRestartLatch) SetReceiver(cancel context.CancelFunc) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.cancelReceiver = cancel
	if l.isRequested {
		cancel()
	}
}

func (l *receiverRestartLatch) ClearReceiver() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.cancelReceiver = nil
}

func (l *receiverRestartLatch) Request() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.isRequested = true
	if l.cancelReceiver != nil {
		l.cancelReceiver()
	}
}

func (l *receiverRestartLatch) Reset() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.isRequested = false
}

func (l *receiverRestartLatch) IsRequested() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.isRequested
}
