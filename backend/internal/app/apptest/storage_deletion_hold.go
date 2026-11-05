package apptest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type HeldStorageDeletion struct {
	Started         <-chan struct{}
	Release         func()
	GetRequestCount func() int64
}

func (namespace StorageNamespace) HoldNextDeletion(t *testing.T, owner Session, shouldFail bool) HeldStorageDeletion {
	t.Helper()

	storage, err := owner.Client.GetStorageWithResponse(t.Context(), namespace.StorageID)
	require.NoError(t, err)
	require.NotNil(t, storage.JSON200)
	settings, err := storage.JSON200.Settings.AsS3StorageSettings()
	require.NoError(t, err)
	upstream, err := url.Parse(*settings.S3Endpoint)
	require.NoError(t, err)

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	started, released := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	var isHeld atomic.Bool
	var requestCount atomic.Int64
	release := func() { releaseOnce.Do(func() { close(released) }) }
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		if request.Method == http.MethodPost && request.URL.Query().Has("delete") &&
			isHeld.CompareAndSwap(false, true) {
			close(started)
			select {
			case <-released:
			case <-t.Context().Done():
				return
			}

			if shouldFail {
				response.Header().Set("Content-Type", "application/xml")
				response.WriteHeader(http.StatusForbidden)
				_, _ = response.Write(
					[]byte(`<Error><Code>AccessDenied</Code><Message>Deletion refused</Message></Error>`),
				)
				return
			}
		}

		proxy.ServeHTTP(response, request)
	}))
	t.Cleanup(server.Close)
	t.Cleanup(release)

	namespace.setEndpoint(t, owner, server.URL)

	return HeldStorageDeletion{Started: started, Release: release, GetRequestCount: requestCount.Load}
}

func (deletion HeldStorageDeletion) WaitUntilStarted(t *testing.T) {
	t.Helper()

	testwait.Until(t, "the provider receives the claimed deletion", func(context.Context) error {
		select {
		case <-deletion.Started:
			return nil
		default:
			return fmt.Errorf("no deletion request yet")
		}
	})
}
