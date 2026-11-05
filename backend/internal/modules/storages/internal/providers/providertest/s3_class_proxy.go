package providertest

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type S3ClassProxy struct {
	URL string

	mutex        sync.Mutex
	writeHeaders []http.Header
}

func StartS3ClassProxy(t *testing.T, targetURL string, isRejectWrites bool) *S3ClassProxy {
	t.Helper()

	target, err := url.Parse(targetURL)
	require.NoError(t, err)

	forwarder := httputil.NewSingleHostReverseProxy(target)
	proxy := &S3ClassProxy{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isWrite := r.Method == http.MethodPut || (r.Method == http.MethodPost && r.URL.Query().Has("uploads"))
		if isWrite {
			proxy.mutex.Lock()
			proxy.writeHeaders = append(proxy.writeHeaders, r.Header.Clone())
			proxy.mutex.Unlock()

			if isRejectWrites {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write(
					[]byte(
						`<Error><Code>InvalidStorageClass</Code><Message>The storage class is not supported</Message></Error>`,
					),
				)
				return
			}
		}

		forwarder.ServeHTTP(w, r)
	}))

	t.Cleanup(server.Close)
	proxy.URL = server.URL

	return proxy
}

func (p *S3ClassProxy) GetWriteHeaders() []http.Header {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	headers := make([]http.Header, len(p.writeHeaders))
	for index, header := range p.writeHeaders {
		headers[index] = header.Clone()
	}

	return headers
}
