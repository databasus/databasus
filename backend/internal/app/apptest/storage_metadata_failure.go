package apptest

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type StorageMetadataFailure struct {
	Started <-chan string
	Fail    func()
	Repair  func()
}

func (namespace StorageNamespace) FailMetadataUploads(
	t *testing.T,
	owner Session,
	prefix string,
) StorageMetadataFailure {
	t.Helper()

	proxy := httputil.NewSingleHostReverseProxy(namespace.Client.EndpointURL())
	started := make(chan string, 1)
	released := make(chan struct{})
	var releaseOnce sync.Once
	var hasStarted atomic.Bool
	var isRepaired atomic.Bool
	release := func() { releaseOnce.Do(func() { close(released) }) }

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		fileName := path.Base(request.URL.Path)
		if request.Method == http.MethodPut && strings.HasPrefix(fileName, prefix) &&
			strings.HasSuffix(fileName, ".metadata") && !isRepaired.Load() {
			if hasStarted.CompareAndSwap(false, true) {
				started <- fileName
				select {
				case <-released:
				case <-t.Context().Done():
					return
				}
			}

			response.Header().Set("Content-Type", "application/xml")
			response.WriteHeader(http.StatusForbidden)
			_, _ = response.Write(
				[]byte(`<Error><Code>AccessDenied</Code><Message>Metadata upload refused</Message></Error>`),
			)

			return
		}

		proxy.ServeHTTP(response, request)
	}))
	t.Cleanup(server.Close)
	t.Cleanup(release)
	namespace.setEndpoint(t, owner, server.URL)

	return StorageMetadataFailure{
		Started: started,
		Fail:    release,
		Repair:  func() { isRepaired.Store(true) },
	}
}
