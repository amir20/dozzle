package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test_previewExpression_respects_user_labels makes sure the notification preview endpoint
// only enumerates containers within the caller's label scope. See GHSA-r9cj-7m9r-h6hq.
func Test_previewExpression_respects_user_labels(t *testing.T) {
	dev := container.Container{ID: "dev123", Name: "dev-allowed", State: "running", Labels: map[string]string{"env": "dev"}}
	prod := container.Container{ID: "prod456", Name: "prod-secret", State: "running", Labels: map[string]string{"env": "prod"}}
	userLabels := container.ContainerLabels{"env": []string{"dev"}}

	client := new(MockedClient)
	client.On("Host").Return(container.Host{ID: "localhost"})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	client.On("ListContainers", mock.Anything, userLabels).Return([]container.Container{dev}, nil)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{dev, prod}, nil)
	client.On("FindContainer", mock.Anything, "dev123").Return(dev, nil)
	client.On("FindContainer", mock.Anything, "prod456").Return(prod, nil)
	client.On("ContainerLogsBetweenDates", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(io.NopCloser(strings.NewReader("")), nil)

	manager := docker_support.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker_support.NewDockerClientService(client, container.ContainerLabels{}))
	h := &handler{
		hostService: docker_support.NewMultiHostService(manager, 3*time.Second),
		config:      &Config{Base: "/", Authorization: Authorization{Provider: SIMPLE}},
	}

	body := strings.NewReader(`{"containerExpression":"true","logExpression":"true"}`)
	req := httptest.NewRequest("POST", "/api/notifications/preview", body)
	req = req.WithContext(auth.WithUser(context.Background(), auth.User{ContainerLabels: userLabels}))
	rr := httptest.NewRecorder()

	h.previewExpression(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var result PreviewResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))

	names := make([]string, 0, len(result.MatchedContainers))
	for _, c := range result.MatchedContainers {
		names = append(names, c.Name)
	}
	assert.Equal(t, []string{"dev-allowed"}, names, "preview should not leak out-of-scope containers")
}
