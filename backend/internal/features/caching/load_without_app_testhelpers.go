//go:build testhelpers

package caching

import (
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/ratelimit"
)

func loadWithoutApp() *installedCaching {
	systemClock := clock.NewSystem()
	store := cache.NewMemoryStore(cache.Deps{PayloadBudgetBytes: cache.DefaultPayloadBudgetBytes, Clock: systemClock})

	return &installedCaching{
		store:       store,
		rateLimiter: ratelimit.NewMemoryCounter(ratelimit.Deps{Store: store, Clock: systemClock}),
	}
}
