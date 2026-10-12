package web

import (
	"net/http"
	"net/http/httptest"

	"testing"
	"testing/fstest"

	"github.com/beme/abide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createRoutes_index(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}

	handler := createHandler(nil, fs, Config{Base: "/", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}

	handler := createHandler(nil, fs, Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/foobar", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("foo page")}}

	handler := createHandler(nil, fs, Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/foobar/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar_file(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": {Data: []byte("index page")},
		"test":       {Data: []byte("test page")},
	}

	handler := createHandler(nil, fs, Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/foobar/test", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Body.String(), "test page", "page doesn't match")
}

func Test_createRoutes_version(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}
	handler := createHandler(nil, fs, Config{Base: "/", Version: "dev", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/api/version", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}
