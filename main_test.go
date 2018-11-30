package main

import (
    "bytes"
    "context"
    "encoding/binary"
    "io"
    "io/ioutil"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"

    "github.com/amir20/dozzle/docker"
    "github.com/beme/abide"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
)

type MockedClient struct {
    mock.Mock
    docker.Client
}

func (m *MockedClient) ListContainers() ([]docker.Container, error) {
    args := m.Called()
    containers, ok := args.Get(0).([]docker.Container)
    if !ok {
        panic("containers is not of type []docker.Container")
    }
    return containers, args.Error(1)
}

func (m *MockedClient) ContainerLogs(ctx context.Context, id string) (io.ReadCloser, error) {
    args := m.Called(ctx, id)
    reader, ok := args.Get(0).(io.ReadCloser)
    if !ok {
        panic("reader is not of type io.ReadCloser")
    }
    return reader, args.Error(1)
}

func Test_handler_listContainers(t *testing.T) {
    req, err := http.NewRequest("GET", "/api/containers.json", nil)
    require.NoError(t, err, "NewRequest should not return an error.")

    rr := httptest.NewRecorder()

    mockedClient := new(MockedClient)
    containers := []docker.Container{
        {
            ID:      "1234567890",
            Status:  "status",
            State:   "state",
            Name:    "test",
            Created: 0,
            Command: "command",
            ImageID: "image_id",
            Image:   "image",
        },
    }
    mockedClient.On("ListContainers", mock.Anything).Return(containers, nil)

    h := handler{client: mockedClient}

    handler := http.HandlerFunc(h.listContainers)

    handler.ServeHTTP(rr, req)
    abide.AssertHTTPResponse(t, "/api/containers.json", rr.Result())
    mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs(t *testing.T) {
    id := "123456"
    req, err := http.NewRequest("GET", "/api/logs/stream", nil)
    q := req.URL.Query()
    q.Add("id", "123456")
    req.URL.RawQuery = q.Encode()
    require.NoError(t, err, "NewRequest should not return an error.")

    rr := httptest.NewRecorder()

    mockedClient := new(MockedClient)
    log := "INFO Testing logs..."
    b := make([]byte, 8)

    binary.BigEndian.PutUint32(b[4:], uint32(len(log)))
    b = append(b, []byte(log)...)

    var reader io.ReadCloser
    reader = ioutil.NopCloser(bytes.NewReader(b))
    mockedClient.On("ContainerLogs", mock.Anything, id).Return(reader, nil)

    h := handler{client: mockedClient}

    handler := http.HandlerFunc(h.streamLogs)

    handler.ServeHTTP(rr, req)
    abide.AssertHTTPResponse(t, "/api/logs/stream", rr.Result())
    mockedClient.AssertExpectations(t)
}

func TestMain(m *testing.M) {
    exit := m.Run()
    abide.Cleanup()
    os.Exit(exit)
}
