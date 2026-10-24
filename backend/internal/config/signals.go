package config

import "sync"

var shutdownSignal = make(chan struct{})

var BeginShutdown = sync.OnceFunc(func() {
	close(shutdownSignal)
})

func GetShutdownSignal() <-chan struct{} {
	return shutdownSignal
}
