package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/clock"
)

func TestJSONStore(t *testing.T) {
	t.Parallel()

	t.Run("keeps a value for the default lifetime unless a lifetime is given", func(t *testing.T) {
		t.Parallel()
		manualClock := clock.NewManual(getTestStart())
		jsonStore := NewJSONStore[string](newTestMemoryStore(1024, manualClock), "values")
		require.NoError(t, jsonStore.Set(t.Context(), "default", "default value"))
		require.NoError(t, jsonStore.SetWithLifetime(t.Context(), ExpiringValue[string]{
			Key:      "custom",
			Value:    "custom value",
			Lifetime: time.Minute,
		}))

		manualClock.Advance(time.Minute)

		customValue, err := jsonStore.Get(t.Context(), "custom")
		require.NoError(t, err)
		assert.Nil(t, customValue)

		defaultValue, err := jsonStore.Get(t.Context(), "default")
		require.NoError(t, err)
		require.NotNil(t, defaultValue)
		assert.Equal(t, "default value", *defaultValue)
	})

	t.Run("keeps equal keys apart when the namespaces differ", func(t *testing.T) {
		t.Parallel()
		memoryStore := newTestMemoryStore(1024, clock.NewManual(getTestStart()))
		firstStore := NewJSONStore[string](memoryStore, "first")
		secondStore := NewJSONStore[string](memoryStore, "second")

		require.NoError(t, firstStore.Set(t.Context(), "shared", "first value"))
		require.NoError(t, secondStore.Set(t.Context(), "shared", "second value"))

		firstValue, err := firstStore.Get(t.Context(), "shared")
		require.NoError(t, err)
		require.NotNil(t, firstValue)
		assert.Equal(t, "first value", *firstValue)

		secondValue, err := secondStore.Get(t.Context(), "shared")
		require.NoError(t, err)
		require.NotNil(t, secondValue)
		assert.Equal(t, "second value", *secondValue)
	})

	t.Run("returns a decode error when the stored payload is malformed", func(t *testing.T) {
		t.Parallel()
		memoryStore := newTestMemoryStore(1024, clock.NewManual(getTestStart()))
		jsonStore := NewJSONStore[string](memoryStore, "values")
		require.NoError(t, memoryStore.Set(t.Context(), Entry{
			Key:      jsonStore.key("malformed"),
			Payload:  []byte("{"),
			Lifetime: time.Minute,
		}))

		value, err := jsonStore.Get(t.Context(), "malformed")

		assert.Nil(t, value)
		require.ErrorContains(t, err, "decode cached value")
	})

	t.Run("returns an encode error when the value cannot be encoded", func(t *testing.T) {
		t.Parallel()
		jsonStore := NewJSONStore[func()](newTestMemoryStore(1024, clock.NewManual(getTestStart())), "functions")

		err := jsonStore.Set(t.Context(), "function", func() {})

		require.ErrorContains(t, err, "encode cached value")
	})

	t.Run("returns the store error when the store fails", func(t *testing.T) {
		t.Parallel()
		storeError := errors.New("store failed")
		jsonStore := NewJSONStore[string](failingStore{err: storeError}, "values")

		_, getError := jsonStore.Get(t.Context(), "key")
		setError := jsonStore.Set(t.Context(), "key", "value")

		require.ErrorIs(t, getError, storeError)
		require.ErrorIs(t, setError, storeError)
	})
}

// failingStore stands in for a store that cannot answer, which the memory store never does.
type failingStore struct {
	err error
}

func (s failingStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Set(context.Context, Entry) error {
	return s.err
}

func (s failingStore) CreateIfAbsent(context.Context, Entry) (bool, error) {
	return false, s.err
}

func (s failingStore) ReadAndDelete(context.Context, string) ([]byte, bool, error) {
	return nil, false, s.err
}

func (s failingStore) Delete(context.Context, string) error {
	return s.err
}
