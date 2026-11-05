package cache_test

import (
	"testing"

	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/cache/cachetest"
	"databasus-backend/internal/lib/clock"
)

func TestMemoryStoreContract(t *testing.T) {
	t.Parallel()

	cachetest.RunStoreContract(t, func(*testing.T) cache.Store {
		return cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: 64 * 1024, Clock: clock.NewSystem()})
	})
}
