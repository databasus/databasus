package browsertests

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gatewaySpec struct {
	backendURL      string
	outputDirectory string
}

func newGateway(t *testing.T, spec gatewaySpec) *httptest.Server {
	t.Helper()
	backend, err := url.Parse(spec.backendURL)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(backend)

	routes := http.NewServeMux()
	routes.Handle("/api/", proxy)
	routes.HandleFunc("/runtime-config.js", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/javascript")
		writer.Header().Set("Cache-Control", "no-store")
		_, _ = writer.Write([]byte(`window.__RUNTIME_CONFIG__ = {IS_EMAIL_CONFIGURED: "true"};`))
	})

	files := http.FileServer(http.Dir(spec.outputDirectory))
	routes.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		candidate := filepath.Join(spec.outputDirectory, filepath.Clean("/"+request.URL.Path))
		if info, statError := os.Stat(candidate); statError == nil && !info.IsDir() {
			files.ServeHTTP(writer, request)
			return
		}
		http.ServeFile(writer, request, filepath.Join(spec.outputDirectory, "index.html"))
	})

	gateway := httptest.NewServer(routes)
	t.Cleanup(gateway.Close)
	return gateway
}

func TestGateway(t *testing.T) {
	t.Parallel()
	t.Run("serves configuration and SPA routes when a production build is mounted", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		require.NoError(
			t,
			os.WriteFile(
				filepath.Join(directory, "index.html"),
				[]byte(`<script src="/runtime-config.js"></script><script type="module" src="/bundle.js"></script>`),
				0o600,
			),
		)

		backend := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "/api/v1/probe", request.URL.Path)
			writer.WriteHeader(http.StatusAccepted)
		}))
		t.Cleanup(backend.Close)
		gateway := newGateway(t, gatewaySpec{backendURL: backend.URL, outputDirectory: directory})
		getResponse := func(address string) (*http.Response, error) {
			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
			require.NoError(t, err)
			return http.DefaultClient.Do(request)
		}

		for _, path := range []string{"/", "/auth/callback", "/storages/google-oauth", "/unknown"} {
			response, err := getResponse(gateway.URL + path)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.NoError(t, response.Body.Close())
		}

		response, err := getResponse(gateway.URL + "/runtime-config.js")
		require.NoError(t, err)
		require.Equal(t, "application/javascript", response.Header.Get("Content-Type"))
		require.NoError(t, response.Body.Close())
		response, err = getResponse(gateway.URL + "/api/v1/probe")
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, response.StatusCode)
		require.NoError(t, response.Body.Close())

		other := newGateway(t, gatewaySpec{backendURL: backend.URL, outputDirectory: directory})
		require.NotEqual(t, gateway.URL, other.URL)
		gateway.Close()

		response, err = getResponse(gateway.URL)
		if response != nil {
			require.NoError(t, response.Body.Close())
		}
		require.Error(t, err)
	})
}
