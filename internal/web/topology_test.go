package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_handler_topology(t *testing.T) {
	containers := []container.Container{
		{
			ID:    "web1",
			Name:  "app-web",
			State: "running",
			Image: "nginx:latest",
			Labels: map[string]string{
				"com.docker.compose.project":    "app",
				"com.docker.compose.service":    "web",
				"com.docker.compose.depends_on": "db:service_started:true",
			},
			Networks: []string{"app_default", "edge"},
			Stats:    utils.NewRingBuffer[container.ContainerStat](300),
		},
		{
			ID:    "db1",
			Name:  "app-db",
			State: "running",
			Image: "postgres:latest",
			Labels: map[string]string{
				"com.docker.compose.project": "app",
				"com.docker.compose.service": "db",
			},
			Networks: []string{"app_default"},
			Stats:    utils.NewRingBuffer[container.ContainerStat](300),
		},
		{
			ID:       "solo1",
			Name:     "standalone",
			State:    "exited",
			Image:    "busybox:latest",
			Labels:   map[string]string{},
			Networks: []string{"bridge"},
			Stats:    utils.NewRingBuffer[container.ContainerStat](300),
		},
	}

	mockedClient := new(MockedClient)
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return(containers, nil)
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	for _, c := range containers {
		mockedClient.On("FindContainer", mock.Anything, c.ID).Return(c, nil)
	}

	handler := createDefaultHandler(mockedClient)
	req, err := http.NewRequest("GET", "/api/topology", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response topologyResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))

	require.Len(t, response.Containers, 3)
	assert.Equal(t, "app-db", response.Containers[0].Name)

	var web *topologyContainer
	for i := range response.Containers {
		if response.Containers[i].ID == "web1" {
			web = &response.Containers[i]
		}
	}
	require.NotNil(t, web)
	assert.Equal(t, "app", web.Stack)
	assert.Equal(t, []string{"app_default", "edge"}, web.Networks)
	assert.Equal(t, []string{"db1"}, web.DependsOn)

	assert.Equal(t, []topologyGroup{
		{Name: "app_default", Containers: []string{"db1", "web1"}},
		{Name: "bridge", Containers: []string{"solo1"}},
		{Name: "edge", Containers: []string{"web1"}},
	}, response.Networks)
	assert.Equal(t, []topologyGroup{{Name: "app", Containers: []string{"db1", "web1"}}}, response.Stacks)
	assert.Equal(t, []topologyEdge{{Source: "web1", Target: "db1"}}, response.DependsOn)
}
