package e2e_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type gatewayOptions struct {
	isClaimHeld  bool
	isStreamHeld bool
	diskCapMb    int
}

type gateway struct {
	url             string
	releaseClaims   func()
	releaseStream   func()
	streamRequested chan struct{}
}

func newGateway(t *testing.T, backendURL string, options gatewayOptions) *gateway {
	t.Helper()
	target, err := url.Parse(backendURL)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	if options.diskCapMb > 0 {
		proxy.ModifyResponse = func(response *http.Response) error {
			if response.StatusCode != http.StatusOK || !strings.HasSuffix(response.Request.URL.Path, "/claim") {
				return nil
			}

			var claim map[string]json.RawMessage
			var body io.Reader = response.Body
			if response.Header.Get("Content-Encoding") == "gzip" {
				compressed, err := gzip.NewReader(response.Body)
				if err != nil {
					return err
				}
				defer func() { _ = compressed.Close() }()
				body = compressed
			}

			err := json.NewDecoder(body).Decode(&claim)
			_ = response.Body.Close()
			if err != nil {
				return err
			}
			claim["maxContainerDiskMb"] = json.RawMessage(strconv.Itoa(options.diskCapMb))
			contents, err := json.Marshal(claim)
			if err != nil {
				return err
			}

			response.Header.Del("Content-Encoding")
			response.Body = io.NopCloser(bytes.NewReader(contents))
			response.ContentLength = int64(len(contents))
			response.Header.Set("Content-Length", strconv.Itoa(len(contents)))
			return nil
		}
	}

	claims := make(chan struct{})
	stream := make(chan struct{})
	var claimOnce, streamOnce, requestedOnce sync.Once
	gate := &gateway{
		url:             "",
		releaseClaims:   func() { claimOnce.Do(func() { close(claims) }) },
		releaseStream:   func() { streamOnce.Do(func() { close(stream) }) },
		streamRequested: make(chan struct{}),
	}

	if !options.isClaimHeld {
		gate.releaseClaims()
	}
	if !options.isStreamHeld {
		gate.releaseStream()
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var release <-chan struct{}
		if strings.HasSuffix(request.URL.Path, "/claim") && request.Method == http.MethodPost {
			release = claims
		}
		if strings.HasSuffix(request.URL.Path, "/backup-stream") && request.Method == http.MethodGet {
			requestedOnce.Do(func() { close(gate.streamRequested) })
			release = stream
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

	gate.url = server.URL
	t.Cleanup(server.Close)
	return gate
}
