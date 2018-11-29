package main

import (
    "github.com/amir20/dozzle/docker"
    "github.com/beme/abide"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
)

type MockedClient struct {
    mock.Mock
    docker.Client
}

func (m *MockedClient) ListContainers() ([]docker.Container, error) {
    args := m.Called()
    containers, _ := args.Get(0).([]docker.Container)
    return containers, args.Error(1)
}

func Test_listContainers(t *testing.T) {
    req, err := http.NewRequest("GET", "/health-check", nil)
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
}

func TestMain(m *testing.M) {
    exit := m.Run()
    abide.Cleanup()
    os.Exit(exit)
}
