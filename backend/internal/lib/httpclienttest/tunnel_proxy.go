package httpclienttest

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

type RecordedRequest struct {
	Method string
	Host   string
	Header http.Header
}

// TunnelProxy is an HTTP proxy that only tunnels: it answers CONNECT and refuses every other
// method, so a test can see exactly what a client hands to a proxy on the way to an https endpoint.
type TunnelProxy struct {
	URL *url.URL

	mutex            sync.Mutex
	receivedRequests []RecordedRequest
}

func StartTunnelProxy(t *testing.T) *TunnelProxy {
	t.Helper()

	proxy := &TunnelProxy{}

	server := httptest.NewServer(http.HandlerFunc(proxy.serve))
	t.Cleanup(server.Close)

	proxyURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse tunnel proxy URL: %v", err)
	}

	proxy.URL = proxyURL

	return proxy
}

func (p *TunnelProxy) GetReceivedRequests() []RecordedRequest {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return append([]RecordedRequest(nil), p.receivedRequests...)
}

func (p *TunnelProxy) serve(w http.ResponseWriter, r *http.Request) {
	p.record(r)

	if r.Method != http.MethodConnect {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	targetConnection, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(r.Context(), "tcp", r.Host)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	hijacker, isHijacker := w.(http.Hijacker)
	if !isHijacker {
		_ = targetConnection.Close()
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	clientConnection, _, err := hijacker.Hijack()
	if err != nil {
		_ = targetConnection.Close()
		return
	}

	if _, err := clientConnection.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = clientConnection.Close()
		_ = targetConnection.Close()

		return
	}

	go pipeAndClose(targetConnection, clientConnection)
	go pipeAndClose(clientConnection, targetConnection)
}

func (p *TunnelProxy) record(r *http.Request) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.receivedRequests = append(p.receivedRequests, RecordedRequest{
		Method: r.Method,
		Host:   r.Host,
		Header: r.Header.Clone(),
	})
}

func pipeAndClose(destination, source net.Conn) {
	_, _ = io.Copy(destination, source)
	_ = destination.Close()
	_ = source.Close()
}

// RecordingTLSServer is an https endpoint with a certificate no client trusts by default, which
// records the requests that reach it and answers each with 200 OK.
type RecordingTLSServer struct {
	URL string

	mutex            sync.Mutex
	receivedRequests []RecordedRequest
}

func StartRecordingTLSServer(t *testing.T) *RecordingTLSServer {
	t.Helper()

	recordingServer := &RecordingTLSServer{}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordingServer.mutex.Lock()
		recordingServer.receivedRequests = append(recordingServer.receivedRequests, RecordedRequest{
			Method: r.Method,
			Host:   r.Host,
			Header: r.Header.Clone(),
		})
		recordingServer.mutex.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	recordingServer.URL = server.URL

	return recordingServer
}

func (s *RecordingTLSServer) GetReceivedRequests() []RecordedRequest {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return append([]RecordedRequest(nil), s.receivedRequests...)
}

func (s *RecordingTLSServer) GetHostPort() string {
	serverURL, err := url.Parse(s.URL)
	if err != nil {
		panic(err)
	}

	return serverURL.Host
}

type ProxiedExchange struct {
	ProxyRequests    []RecordedRequest
	EndpointRequests []RecordedRequest
	EndpointHostPort string
	Err              error
}

// SendThroughTunnelProxy replaces only the transport's Proxy with a fresh local tunnel proxy and
// sends one request carrying the given Authorization header to a fresh untrusted https endpoint.
// The transport keeps every other setting the code under test gave it, TLS included.
func SendThroughTunnelProxy(t *testing.T, transport *http.Transport, authorization string) ProxiedExchange {
	t.Helper()

	proxy := StartTunnelProxy(t)
	endpoint := StartRecordingTLSServer(t)

	transport.Proxy = http.ProxyURL(proxy.URL)
	defer transport.CloseIdleConnections()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, endpoint.URL+"/object", nil)
	if err != nil {
		t.Fatalf("build proxied request: %v", err)
	}

	request.Header.Set("Authorization", authorization)

	response, requestErr := (&http.Client{Transport: transport}).Do(request)
	if requestErr == nil {
		_ = response.Body.Close()
	}

	return ProxiedExchange{
		ProxyRequests:    proxy.GetReceivedRequests(),
		EndpointRequests: endpoint.GetReceivedRequests(),
		EndpointHostPort: endpoint.GetHostPort(),
		Err:              requestErr,
	}
}

func AssertProxyReceivedOnlyTunnelRequest(t *testing.T, exchange ProxiedExchange) {
	t.Helper()

	if len(exchange.ProxyRequests) != 1 {
		t.Fatalf("expected exactly one request at the proxy, got %d", len(exchange.ProxyRequests))
	}

	proxyRequest := exchange.ProxyRequests[0]

	if proxyRequest.Method != http.MethodConnect {
		t.Errorf("expected the proxy to receive CONNECT, got %s", proxyRequest.Method)
	}

	if proxyRequest.Host != exchange.EndpointHostPort {
		t.Errorf("expected CONNECT to %s, got %s", exchange.EndpointHostPort, proxyRequest.Host)
	}

	if proxyRequest.Header.Get("Authorization") != "" {
		t.Error("the proxy received the endpoint's Authorization header")
	}
}

func AssertCertificateRejected(t *testing.T, exchange ProxiedExchange) {
	t.Helper()

	if _, isCertificateVerificationError := errors.AsType[*tls.CertificateVerificationError](
		exchange.Err,
	); !isCertificateVerificationError {
		t.Fatalf("expected a certificate verification error, got %v", exchange.Err)
	}

	if len(exchange.EndpointRequests) != 0 {
		t.Error("a request reached the endpoint despite its untrusted certificate")
	}
}
