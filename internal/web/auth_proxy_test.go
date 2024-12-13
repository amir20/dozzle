package web

import (
	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
)

func Test_createRoutes_proxy_missing_headers(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/",
		Authorization: Authorization{
			Provider:   FORWARD_PROXY,
			Authorizer: auth.NewForwardProxyAuth("Remote-User", "Remote-Email", "Remote-Name", "Remote-Filter"),
		},
	})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, 401, rr.Code, "Response code should be 401.")
}

func Test_createRoutes_proxy_happy(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/",
		Authorization: Authorization{
			Provider:   FORWARD_PROXY,
			Authorizer: auth.NewForwardProxyAuth("Remote-User", "Remote-Email", "Remote-Name", "Remote-Filter"),
		},
	})
	req, err := http.NewRequest("GET", "/", nil)
	req.Header.Set("Remote-Email", "amir@test.com")
	req.Header.Set("Remote-Name", "Amir")
	req.Header.Set("Remote-User", "amir")
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, 200, rr.Code, "Response code should be 200.")
}
