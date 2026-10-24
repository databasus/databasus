package config

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_BeginShutdown_WhenCalledTwiceConcurrently_ClosesShutdownSignalOnce(t *testing.T) {
	var shutdownCallers sync.WaitGroup

	for range 2 {
		shutdownCallers.Go(func() {
			assert.NotPanics(t, BeginShutdown)
		})
	}

	shutdownCallers.Wait()

	select {
	case <-GetShutdownSignal():
	case <-time.After(time.Second):
		t.Fatal("shutdown signal was not closed")
	}
}
