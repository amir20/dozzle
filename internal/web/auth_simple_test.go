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

	"github.com/amir20/dozzle/internal/auth"
	"github.com/beme/abide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
)

func Test_createRoutes_simple_redirect(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
					},
				},
			}, time.Second*100),
		},
	})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_simple_valid_token(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
					},
				},
			}, time.Second*100),
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
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/",
		Authorization: Authorization{
			Provider: SIMPLE,
			Authorizer: auth.NewSimpleAuth(auth.UserDatabase{
				Users: map[string]*auth.User{
					"amir": {
						Username: "amir",
						Password: "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
					},
				},
			}, time.Second*100),
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
