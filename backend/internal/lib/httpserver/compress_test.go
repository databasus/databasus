package httpserver

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompress(t *testing.T) {
	t.Parallel()

	for _, responseCase := range []struct {
		name           string
		path           string
		acceptEncoding string
		contentType    string
		body           string
		shouldCompress bool
	}{
		{
			name: "compresses a json response when the client accepts gzip", path: "/records", acceptEncoding: "gzip",
			contentType: "application/json", body: `{"status":"ready"}`, shouldCompress: true,
		},
		{
			name: "compresses javascript when the client accepts gzip", path: "/assets/app.js", acceptEncoding: "gzip",
			contentType: "text/javascript", body: `window.status = "ready";`, shouldCompress: true,
		},
		{
			name: "compresses css when the client accepts gzip", path: "/assets/app.css", acceptEncoding: "gzip",
			contentType: "text/css", body: "body { color: black; }", shouldCompress: true,
		},
		{
			name: "preserves a json response when the client requests identity encoding", path: "/records", acceptEncoding: "identity",
			contentType: "application/json", body: `{"status":"ready"}`, shouldCompress: false,
		},
		{
			name: "preserves a json response when the client omits accepted encodings", path: "/records", acceptEncoding: "",
			contentType: "application/json", body: `{"status":"ready"}`, shouldCompress: false,
		},
		{
			name: "preserves css when the client requests identity encoding", path: "/assets/app.css", acceptEncoding: "identity",
			contentType: "text/css", body: "body { color: black; }", shouldCompress: false,
		},
		{
			name: "preserves a stream when its path is excluded", path: "/exports/one/file", acceptEncoding: "gzip",
			contentType: "application/octet-stream", body: "file bytes", shouldCompress: false,
		},
		{
			name: "preserves an excluded stream when its url has a query", path: "/exports/one/file?token=download", acceptEncoding: "gzip",
			contentType: "application/octet-stream", body: "file bytes", shouldCompress: false,
		},
		{
			name: "compresses a neighboring response when its path shares an excluded prefix", path: "/exports/one/file-info", acceptEncoding: "gzip",
			contentType: "application/json", body: `{"status":"ready"}`, shouldCompress: true,
		},
		{
			name: "compresses a neighboring response when its path extends an excluded path", path: "/exports/one/file/details", acceptEncoding: "gzip",
			contentType: "application/json", body: `{"status":"ready"}`, shouldCompress: true,
		},
		{
			name: "compresses a neighboring response when its path prefixes an excluded path", path: "/other/exports/one/file", acceptEncoding: "gzip",
			contentType: "application/json", body: `{"status":"ready"}`, shouldCompress: true,
		},
	} {
		t.Run(responseCase.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			router.Use(Compress([]string{`^/exports/[^/]+/file$`}))
			router.NoRoute(func(ctx *gin.Context) {
				ctx.Data(http.StatusOK, responseCase.contentType, []byte(responseCase.body))
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, responseCase.path, nil)
			request.Header.Set("Accept-Encoding", responseCase.acceptEncoding)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)

			if !responseCase.shouldCompress {
				assert.Empty(t, response.Header().Get("Content-Encoding"))
				assert.Equal(t, responseCase.body, response.Body.String())
				return
			}

			require.Equal(t, "gzip", response.Header().Get("Content-Encoding"))
			assert.Contains(t, response.Header().Values("Vary"), "Accept-Encoding")

			decoder, err := gzip.NewReader(bytes.NewReader(response.Body.Bytes()))
			require.NoError(t, err)
			decompressedBody, err := io.ReadAll(decoder)
			require.NoError(t, err)
			require.NoError(t, decoder.Close())

			assert.Equal(t, responseCase.body, string(decompressedBody))
		})
	}

	for _, extension := range []string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"} {
		t.Run("preserves an asset when the excluded extension is "+extension, func(t *testing.T) {
			t.Parallel()

			const assetBody = "asset bytes"
			router := gin.New()
			router.Use(Compress(nil))
			router.GET("/assets/file"+extension, func(ctx *gin.Context) {
				ctx.Data(http.StatusOK, "application/octet-stream", []byte(assetBody))
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/file"+extension, nil)
			request.Header.Set("Accept-Encoding", "gzip")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assert.Equal(t, http.StatusOK, response.Code)
			assert.Empty(t, response.Header().Get("Content-Encoding"))
			assert.Equal(t, assetBody, response.Body.String())
		})
	}
}
