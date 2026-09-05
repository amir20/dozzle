package web

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const assetJS = "export const answer = 42;\n"

// Text assets ship only as the `.br` sibling written by scripts/compress-dist.js, so
// the handler negotiates the encoding itself rather than leaning on http.FileServer.
func assetHandler(t *testing.T) *handler {
	t.Helper()

	var buf bytes.Buffer
	bw := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	_, err := bw.Write([]byte(assetJS))
	require.NoError(t, err)
	require.NoError(t, bw.Close())

	h := &handler{
		content: fstest.MapFS{
			"assets/app-abc12345.js.br":  {Data: buf.Bytes()},
			"assets/font-abc12345.woff2": {Data: []byte("not compressible")},
		},
		config: &Config{},
	}
	fileServer = http.FileServer(http.FS(h.content))
	return h
}

// index() runs behind http.StripPrefix, so it sees paths without a leading slash.
func assetRequest(name string, acceptEncoding string) *http.Request {
	req := httptest.NewRequest("GET", "/"+name, nil)
	req.URL.Path = name
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	return req
}

func TestServeAsset(t *testing.T) {
	h := assetHandler(t)

	t.Run("serves the precompressed bytes to a brotli client", func(t *testing.T) {
		req := assetRequest("assets/app-abc12345.js", "gzip, deflate, br, zstd")
		w := httptest.NewRecorder()

		require.True(t, h.serveAsset(w, req, req.URL.Path))
		res := w.Result()

		assert.Equal(t, "br", res.Header.Get("Content-Encoding"))
		assert.Equal(t, "text/javascript; charset=utf-8", res.Header.Get("Content-Type"))
		assert.Equal(t, "Accept-Encoding", res.Header.Get("Vary"))
		assert.NotEmpty(t, res.Header.Get("Content-Length"))

		body, err := io.ReadAll(brotli.NewReader(res.Body))
		require.NoError(t, err)
		assert.Equal(t, assetJS, string(body))
	})

	t.Run("inflates for a client that does not accept brotli", func(t *testing.T) {
		req := assetRequest("assets/app-abc12345.js", "")
		w := httptest.NewRecorder()

		require.True(t, h.serveAsset(w, req, req.URL.Path))
		res := w.Result()

		assert.Empty(t, res.Header.Get("Content-Encoding"))
		assert.Equal(t, "text/javascript; charset=utf-8", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, assetJS, string(body))
	})

	t.Run("serves already-compressed assets untouched", func(t *testing.T) {
		req := assetRequest("assets/font-abc12345.woff2", "br")
		w := httptest.NewRecorder()

		require.True(t, h.serveAsset(w, req, req.URL.Path))
		res := w.Result()

		assert.Empty(t, res.Header.Get("Content-Encoding"))
		body, _ := io.ReadAll(res.Body)
		assert.Equal(t, "not compressible", string(body))
	})

	t.Run("falls through so unknown paths reach the SPA template", func(t *testing.T) {
		req := assetRequest("container/abc", "br")
		assert.False(t, h.serveAsset(httptest.NewRecorder(), req, req.URL.Path))
	})
}
