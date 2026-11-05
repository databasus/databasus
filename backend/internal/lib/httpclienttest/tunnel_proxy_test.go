package httpclienttest

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SendThroughTunnelProxy_WhenVerificationSkipped_ReachesEndpointThroughOneConnect(t *testing.T) {
	exchange := SendThroughTunnelProxy(t, &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}, "secret-signature")

	require.NoError(t, exchange.Err)
	AssertProxyReceivedOnlyTunnelRequest(t, exchange)
	require.Len(t, exchange.EndpointRequests, 1)
	assert.Equal(t, "secret-signature", exchange.EndpointRequests[0].Header.Get("Authorization"))
}

func Test_SendThroughTunnelProxy_WhenVerificationOn_RejectsEndpointCertificate(t *testing.T) {
	exchange := SendThroughTunnelProxy(t, &http.Transport{}, "secret-signature")

	AssertProxyReceivedOnlyTunnelRequest(t, exchange)
	AssertCertificateRejected(t, exchange)
}

func Test_GetNetworkTimeoutError_ReturnsErrorMatchingDeadlineExceeded(t *testing.T) {
	timeoutErr := GetNetworkTimeoutError(t)

	assert.ErrorIs(t, timeoutErr, context.DeadlineExceeded)
}
