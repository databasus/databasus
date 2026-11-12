package providertest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type RefusingProxySpec struct {
	TargetURL string

	IsRefusable  func(r *http.Request) bool
	RefusalCount int

	Status      int
	ContentType string
	Body        string
}

// RefusingProxy fronts a real storage with refusals it cannot produce on its own, such as a busy
// node answering 503: the first RefusalCount refusable requests get the refusal, and every other
// request passes through unchanged. The incoming Host header is kept, so request signatures still
// match at the storage.
type RefusingProxy struct {
	URL string

	refusableCount atomic.Int32
}

func StartRefusingProxy(t *testing.T, spec RefusingProxySpec) *RefusingProxy {
	t.Helper()

	target, err := url.Parse(spec.TargetURL)
	require.NoError(t, err)

	forwarder := httputil.NewSingleHostReverseProxy(target)
	proxy := &RefusingProxy{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !spec.IsRefusable(r) || int(proxy.refusableCount.Add(1)) > spec.RefusalCount {
			forwarder.ServeHTTP(w, r)
			return
		}

		_, _ = io.Copy(io.Discard, r.Body)

		w.Header().Set("Content-Type", spec.ContentType)
		w.WriteHeader(spec.Status)
		_, _ = w.Write([]byte(spec.Body))
	}))

	t.Cleanup(server.Close)
	proxy.URL = server.URL

	return proxy
}

// GetRefusableCount counts every request that matched IsRefusable, refused or passed through.
func (p *RefusingProxy) GetRefusableCount() int {
	return int(p.refusableCount.Load())
}
