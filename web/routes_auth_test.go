package web

import (
	"bytes"

	"io"

	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"strings"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"

	"github.com/amir20/dozzle/docker"
	"github.com/beme/abide"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
)

func Test_createRoutes_index(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/"})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar"})
	req, err := http.NewRequest("GET", "/foobar", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect_with_auth(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar", Username: "amir", Password: "password"})
	req, err := http.NewRequest("GET", "/foobar/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("foo page"), 0644), "WriteFile should have no error.")
	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar"})
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

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar"})
	req, err := http.NewRequest("GET", "/foobar/test", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Body.String(), "test page", "page doesn't match")
}

func Test_createRoutes_version(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/", Version: "dev"})
	req, err := http.NewRequest("GET", "/version", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password(t *testing.T) {

	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password"})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password_invalid(t *testing.T) {
	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password"})
	req, err := http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password_login_happy(t *testing.T) {
	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password"})

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

	req, err := http.NewRequest("POST", "/api/validateCredentials", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, 200)
	cookie := rr.Header().Get("Set-Cookie")
	assert.Matches(t, cookie, "session=.+")
}

func Test_createRoutes_username_password_login_failed(t *testing.T) {
	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password"})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormField("username")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("amir"))
	require.NoError(t, err, "Copying field should not result in error.")

	fw, err = writer.CreateFormField("password")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("bad"))
	require.NoError(t, err, "Copying field should not result in error.")

	writer.Close()

	req, err := http.NewRequest("POST", "/api/validateCredentials", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 401)
}

func Test_createRoutes_username_password_valid_session(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", "123").Return(docker.Container{ID: "123"}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, "123", "").Return(ioutil.NopCloser(strings.NewReader("test data")), io.EOF)
	handler := createHandler(mockedClient, nil, Config{Base: "/", Username: "amir", Password: "password"})

	// Get cookie first
	req, err := http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	session, _ := store.Get(req, sessionName)
	session.Values[authorityKey] = time.Now().Unix()
	recorder := httptest.NewRecorder()
	session.Save(req, recorder)
	cookies := recorder.Result().Cookies()

	// Test with cookie
	req, err = http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	req.AddCookie(cookies[0])
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password_invalid_session(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", "123").Return(docker.Container{ID: "123"}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, "since").Return(ioutil.NopCloser(strings.NewReader("test data")), io.EOF)
	handler := createHandler(mockedClient, nil, Config{Base: "/", Username: "amir", Password: "password"})
	req, err := http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	req.AddCookie(&http.Cookie{Name: "session", Value: "baddata"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 401)
}
