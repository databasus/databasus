//go:build !testhelpers

package caching

func loadWithoutApp() *installedCaching {
	panic("caching.GetStore or GetRateLimiter is called before app.New installed the cache")
}
