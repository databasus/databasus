package providertest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefusingProxy(t *testing.T) {
	t.Parallel()

	t.Run("refuses the first matching requests and passes the rest through", func(t *testing.T) {
		t.Parallel()

		storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("stored"))
		}))
		t.Cleanup(storage.Close)

		proxy := StartRefusingProxy(t, RefusingProxySpec{
			TargetURL:    storage.URL,
			IsRefusable:  func(r *http.Request) bool { return r.Method == http.MethodPut },
			RefusalCount: 2,
			Status:       http.StatusServiceUnavailable,
			ContentType:  "application/xml",
			Body:         "<Error><Code>ServiceUnavailable</Code></Error>",
		})

		firstPutStatus, firstPutBody := sendThroughProxy(t, http.MethodPut, proxy.URL)
		getStatus, getBody := sendThroughProxy(t, http.MethodGet, proxy.URL)
		secondPutStatus, _ := sendThroughProxy(t, http.MethodPut, proxy.URL)
		thirdPutStatus, thirdPutBody := sendThroughProxy(t, http.MethodPut, proxy.URL)

		assert.Equal(t, http.StatusServiceUnavailable, firstPutStatus)
		assert.Equal(t, "<Error><Code>ServiceUnavailable</Code></Error>", firstPutBody)
		assert.Equal(t, http.StatusOK, getStatus)
		assert.Equal(t, "stored", getBody)
		assert.Equal(t, http.StatusServiceUnavailable, secondPutStatus)
		assert.Equal(t, http.StatusOK, thirdPutStatus)
		assert.Equal(t, "stored", thirdPutBody)
		assert.Equal(t, 3, proxy.GetRefusableCount())
	})
}

func sendThroughProxy(t *testing.T, method, proxyURL string) (int, string) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, proxyURL+"/object", strings.NewReader("payload"))
	require.NoError(t, err)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return response.StatusCode, string(body)
}
