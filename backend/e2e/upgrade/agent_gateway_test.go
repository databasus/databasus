package upgrade

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type agentGateway struct {
	url               string
	rejectedHeartbeat chan struct{}
	acceptedHeartbeat chan struct{}
	releaseDiscovery  func()
	releaseClaims     func()
}

func newAgentGateway(t *testing.T, api *apiClient) *agentGateway {
	t.Helper()

	target, err := url.Parse(strings.TrimSuffix(api.baseURL, "/api/v1"))
	require.NoError(t, err)

	proxy := httputil.NewSingleHostReverseProxy(target)
	discovery, claims := make(chan struct{}), make(chan struct{})
	var discoveryOnce, claimsOnce, rejectedOnce, acceptedOnce sync.Once
	gateway := &agentGateway{
		rejectedHeartbeat: make(chan struct{}),
		acceptedHeartbeat: make(chan struct{}),
		releaseDiscovery:  func() { discoveryOnce.Do(func() { close(discovery) }) },
		releaseClaims:     func() { claimsOnce.Do(func() { close(claims) }) },
	}

	proxy.ModifyResponse = func(response *http.Response) error {
		if strings.HasSuffix(response.Request.URL.Path, "/heartbeat") {
			switch response.StatusCode {
			case http.StatusBadRequest:
				rejectedOnce.Do(func() { close(gateway.rejectedHeartbeat) })
			case http.StatusOK:
				acceptedOnce.Do(func() { close(gateway.acceptedHeartbeat) })
			}
		}

		return nil
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var release <-chan struct{}
		if request.URL.Path == "/api/v1/system/version" {
			release = discovery
		}

		if strings.HasSuffix(request.URL.Path, "/claim") {
			release = claims
		}

		if release != nil {
			select {
			case <-release:
			case <-request.Context().Done():
				return
			case <-t.Context().Done():
				return
			}
		}

		proxy.ServeHTTP(writer, request)
	}))
	gateway.url = server.URL

	t.Cleanup(func() { gateway.releaseDiscovery(); gateway.releaseClaims(); server.Close() })

	return gateway
}
