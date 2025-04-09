package web

import (
	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/beme/abide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
)

func Test_createRoutes_index(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/foobar", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("foo page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/foobar/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar_file(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	require.NoError(t, afero.WriteFile(fs, "test", []byte("test page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/foobar/test", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Body.String(), "test page", "page doesn't match")
}

func Test_createRoutes_version(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/", Version: "dev", Authorization: Authorization{Provider: NONE}})
	req, err := http.NewRequest("GET", "/api/version", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}
