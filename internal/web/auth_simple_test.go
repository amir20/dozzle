package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"testing"
	"testing/fstest"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSecret stands in for the persisted secret from auth.SessionSecret, fixed so
// the snapshotted responses stay stable.
var testSecret = []byte("test-session-secret")

func Test_createRoutes_simple_redirect(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}

	handler := createHandler(nil, fs, Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "$2a$04$gma4F9WIu3g1ILzyj5Xvv.3sBkbmX20wDHNE.1oNw0mJvB9G.Vb0C",
					},
				},
			}, time.Second*100, testSecret),
		},
	})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	matchHTTPSnapshot(t, rr.Result())
}

func Test_createRoutes_simple_valid_token(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}

	handler := createHandler(nil, fs, Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "$2a$04$gma4F9WIu3g1ILzyj5Xvv.3sBkbmX20wDHNE.1oNw0mJvB9G.Vb0C",
					},
				},
			}, time.Second*100, testSecret),
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormField("username")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("amir"))
	require.NoError(t, err, "Copying field should not result in error.")

	fw, err = writer.CreateFormField("password")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("password"))
	require.NoError(t, err, "Copying field should not result in error.")

	writer.Close()

	req, err := http.NewRequest("POST", "/api/token", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, 200)
	cookie := rr.Header().Get("Set-Cookie")
	assert.Regexp(t, `jwt=.+`, cookie)
}

func Test_createRoutes_simple_bad_password(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}

	handler := createHandler(nil, fs, Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "$2a$04$gma4F9WIu3g1ILzyj5Xvv.3sBkbmX20wDHNE.1oNw0mJvB9G.Vb0C",
					},
				},
			}, time.Second*100, testSecret),
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormField("username")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("amir"))
	require.NoError(t, err, "Copying field should not result in error.")

	fw, err = writer.CreateFormField("badpassword")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("password"))
	require.NoError(t, err, "Copying field should not result in error.")

	writer.Close()

	req, err := http.NewRequest("POST", "/api/token", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, 401, "Response code should be 401.")
}

func Test_createRoutes_simple_cloud_callback_requires_auth(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}

	handler := createHandler(nil, fs, Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "$2a$04$gma4F9WIu3g1ILzyj5Xvv.3sBkbmX20wDHNE.1oNw0mJvB9G.Vb0C",
					},
				},
			}, time.Second*100, testSecret),
		},
	})

	// The cloud callback must not be reachable without authentication. Otherwise
	// an attacker could force-link the instance to their own cloud account.
	for _, path := range []string{"/api/cloud/callback?token=attacker", "/api/cloud/status"} {
		req, err := http.NewRequest("GET", path, nil)
		require.NoError(t, err, "NewRequest should not return an error.")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, 401, rr.Code, "%s should require authentication.", path)
	}
}

// A back swipe after signing in lands on /login with a live session. Rendering the
// form again would ask for credentials the browser already holds.
func Test_createRoutes_simple_login_page_redirects_signed_in_user(t *testing.T) {
	fs := fstest.MapFS{"index.html": {Data: []byte("index page")}}

	handler := createHandler(nil, fs, Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "$2a$04$gma4F9WIu3g1ILzyj5Xvv.3sBkbmX20wDHNE.1oNw0mJvB9G.Vb0C",
					},
				},
			}, time.Second*100, testSecret),
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("username", "amir"))
	require.NoError(t, writer.WriteField("password", "password"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/token", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	cookies := rr.Result().Cookies()
	require.NotEmpty(t, cookies)

	get := func(target string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", target, nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr
	}

	rr = get("/login?redirectUrl=%2Fcontainer%2Fabc")
	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	assert.Equal(t, "/container/abc", rr.Header().Get("Location"))

	rr = get("/login")
	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))

	rr = get("/login?redirectUrl=%2F%2Fevil.com")
	assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))
}
