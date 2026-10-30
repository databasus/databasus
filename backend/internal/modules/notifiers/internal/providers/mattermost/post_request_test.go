package mattermost

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	test_utils "databasus-backend/internal/lib/httpclienttest"
)

func TestNewTLSSkippingTransport(t *testing.T) {
	t.Parallel()

	t.Run("takes the proxy from the environment and keeps the TLS settings", func(t *testing.T) {
		t.Parallel()

		transport := newTLSSkippingTransport()

		require.NotNil(t, transport.Proxy)
		assert.Equal(t,
			reflect.ValueOf(http.ProxyFromEnvironment).Pointer(),
			reflect.ValueOf(transport.Proxy).Pointer(),
		)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
		assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
	})

	t.Run("lets the credentials reach only the endpoint when it goes through a proxy", func(t *testing.T) {
		t.Parallel()

		exchange := test_utils.SendThroughTunnelProxy(t, newTLSSkippingTransport(), "Bearer bot-token")

		require.NoError(t, exchange.Err)
		test_utils.AssertProxyReceivedOnlyTunnelRequest(t, exchange)
		require.Len(t, exchange.EndpointRequests, 1)
		assert.Equal(t, "Bearer bot-token", exchange.EndpointRequests[0].Header.Get("Authorization"))
	})
}
