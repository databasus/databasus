// Package caching hands the cache store and the rate limiter that internal/app built to the code
// that has not moved into internal/modules yet. A test binary built with the testhelpers tag builds
// them on first use instead.
package caching

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/ratelimit"
)

type installedCaching struct {
	store       cache.Store
	rateLimiter ratelimit.Counter
}

var installed atomic.Pointer[installedCaching]

var getCachingWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(store cache.Store, rateLimiter ratelimit.Counter) {
	installed.Store(&installedCaching{store: store, rateLimiter: rateLimiter})
}

func GetStore() cache.Store {
	return getCaching().store
}

func GetRateLimiter() ratelimit.Counter {
	return getCaching().rateLimiter
}

func getCaching() *installedCaching {
	if caching := installed.Load(); caching != nil {
		return caching
	}

	return getCachingWithoutApp()
}
