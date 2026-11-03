// Package cachetest checks that a cache store keeps the promises of cache.Store.
package cachetest

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/cache"
)

type StoreFactory func(t *testing.T) cache.Store

func RunStoreContract(t *testing.T, createStore StoreFactory) {
	t.Helper()

	t.Run("copies payloads when an entry is set and read", func(t *testing.T) {
		t.Parallel()
		store := createStore(t)
		sourcePayload := []byte("value")
		require.NoError(t, store.Set(t.Context(), cache.Entry{
			Key: "entry", Payload: sourcePayload, Lifetime: time.Minute,
		}))
		sourcePayload[0] = 'X'

		storedPayload, isFound, err := store.Get(t.Context(), "entry")
		require.NoError(t, err)
		require.True(t, isFound)
		assert.Equal(t, []byte("value"), storedPayload)
		storedPayload[0] = 'X'

		storedPayload, isFound, err = store.Get(t.Context(), "entry")
		require.NoError(t, err)
		require.True(t, isFound)
		assert.Equal(t, []byte("value"), storedPayload)
	})

	t.Run("returns an error when the context is canceled", func(t *testing.T) {
		t.Parallel()
		store := createStore(t)
		canceledContext, cancel := context.WithCancel(t.Context())
		cancel()

		err := store.Set(canceledContext, cache.Entry{
			Key: "entry", Payload: []byte("value"), Lifetime: time.Minute,
		})
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("creates the entry once when callers race to create it", func(t *testing.T) {
		t.Parallel()
		store := createStore(t)
		const callerCount = 32
		var createdCount atomic.Int64
		operationErrors := make(chan error, callerCount)
		var waitGroup sync.WaitGroup

		for callerIndex := range callerCount {
			waitGroup.Go(func() {
				isCreated, err := store.CreateIfAbsent(t.Context(), cache.Entry{
					Key: "reservation", Payload: []byte{byte(callerIndex)}, Lifetime: time.Minute,
				})
				operationErrors <- err
				if isCreated {
					createdCount.Add(1)
				}
			})
		}

		waitGroup.Wait()
		close(operationErrors)

		for operationError := range operationErrors {
			require.NoError(t, operationError)
		}

		assert.Equal(t, int64(1), createdCount.Load())
	})

	t.Run("hands the value to one caller when callers race to read and delete it", func(t *testing.T) {
		t.Parallel()
		store := createStore(t)
		require.NoError(t, store.Set(t.Context(), cache.Entry{
			Key: "token", Payload: []byte("secret"), Lifetime: time.Minute,
		}))

		const callerCount = 32
		var consumedCount atomic.Int64
		operationErrors := make(chan error, callerCount)
		var waitGroup sync.WaitGroup

		for range callerCount {
			waitGroup.Go(func() {
				payload, isFound, err := store.ReadAndDelete(t.Context(), "token")
				operationErrors <- err
				if isFound {
					assert.Equal(t, []byte("secret"), payload)
					consumedCount.Add(1)
				}
			})
		}

		waitGroup.Wait()
		close(operationErrors)

		for operationError := range operationErrors {
			require.NoError(t, operationError)
		}

		assert.Equal(t, int64(1), consumedCount.Load())
	})
}
