package apptest

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type HeldStorageUpload struct {
	Started  <-chan string
	Finished <-chan struct{}
	Release  func()
}

func (namespace StorageNamespace) HoldNextUpload(t *testing.T, owner Session, prefix string) HeldStorageUpload {
	t.Helper()
	storage, err := owner.Client.GetStorageWithResponse(t.Context(), namespace.StorageID)
	require.NoError(t, err)
	require.NotNil(t, storage.JSON200)
	settings, err := storage.JSON200.Settings.AsS3StorageSettings()
	require.NoError(t, err)
	upstream, err := url.Parse(*settings.S3Endpoint)
	require.NoError(t, err)

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	started := make(chan string, 1)
	finished := make(chan struct{})
	released := make(chan struct{})
	var releaseOnce sync.Once
	var isHeld atomic.Bool
	release := func() { releaseOnce.Do(func() { close(released) }) }

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		fileName := path.Base(request.URL.Path)
		if request.Method == http.MethodPut && strings.HasPrefix(fileName, prefix) &&
			!strings.HasSuffix(fileName, ".metadata") &&
			isHeld.CompareAndSwap(false, true) {
			defer close(finished)
			body, readErr := io.ReadAll(request.Body)
			if readErr != nil {
				http.Error(response, "read upload", http.StatusBadRequest)
				return
			}
			request.Body = io.NopCloser(bytes.NewReader(body))
			started <- fileName
			select {
			case <-released:
			case <-t.Context().Done():
				return
			}
		}

		proxy.ServeHTTP(response, request)
	}))
	t.Cleanup(server.Close)
	t.Cleanup(release)

	namespace.setEndpoint(t, owner, server.URL)
	return HeldStorageUpload{Started: started, Finished: finished, Release: release}
}
