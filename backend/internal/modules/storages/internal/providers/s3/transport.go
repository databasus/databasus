package s3

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
)

// Go keeps two idle connections per host by default, which drops connections as soon as more than
// two writes to one bucket run at once.
const s3MaxIdleConnsPerHost = 16

// sharedTransports keep the connections of every S3 storage. Clients are still built per call from
// freshly decrypted credentials. Only the connections are shared: S3 signs every request, so a
// changed key takes effect on the next request even over a connection opened before the change, and
// a changed endpoint lands in its own connection pool.
type sharedTransports struct {
	verifying     *http.Transport
	skippingCheck *http.Transport
}

func newSharedTransports() sharedTransports {
	return sharedTransports{verifying: newSharedTransport(false), skippingCheck: newSharedTransport(true)}
}

func (t sharedTransports) get(isSkipTLSVerify bool) *http.Transport {
	if isSkipTLSVerify {
		return t.skippingCheck
	}

	return t.verifying
}

func newSharedTransport(isSkipTLSVerify bool) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: s3ConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   s3TLSHandshakeTimeout,
		ResponseHeaderTimeout: s3ResponseTimeout,
		IdleConnTimeout:       s3IdleConnTimeout,
		MaxIdleConnsPerHost:   s3MaxIdleConnsPerHost,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: isSkipTLSVerify,
		},
	}
}

type bucketRegionKey struct {
	endpoint string
	bucket   string
}

// bucketRegionCache exists because a client built per call loses minio's own bucket-location cache,
// so without it every write to a storage with no region set would pay an extra location request.
type bucketRegionCache struct {
	mutex   sync.RWMutex
	regions map[bucketRegionKey]string
}

func newBucketRegionCache() *bucketRegionCache {
	return &bucketRegionCache{mutex: sync.RWMutex{}, regions: map[bucketRegionKey]string{}}
}

// getRegion does not cache a failed lookup and answers it with an empty region, so the client falls
// back to minio's own lookup and reports the real error on the request it was built for.
func (c *bucketRegionCache) getRegion(ctx context.Context, params clientParams, bucketName string) string {
	key := bucketRegionKey{endpoint: params.endpoint, bucket: bucketName}

	c.mutex.RLock()
	cachedRegion, isCached := c.regions[key]
	c.mutex.RUnlock()

	if isCached {
		return cachedRegion
	}

	lookupClient, err := newMinioClient(params, "")
	if err != nil {
		return ""
	}

	region, err := lookupClient.GetBucketLocation(ctx, bucketName)
	if err != nil {
		return ""
	}

	c.mutex.Lock()
	c.regions[key] = region
	c.mutex.Unlock()

	return region
}
