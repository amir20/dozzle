package web

import (
	"bytes"
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
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/hostservice"
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

	manager := hostservice.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker.NewService(client, container.ContainerLabels{}))
	h := &handler{
		hostService: hostservice.NewMultiHostService(manager, 3*time.Second),
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

// The alert form renders preview matches with the same v-html log items as the
// live stream, so they must come back escaped like the stream does.
func Test_previewExpression_escapes_matched_logs(t *testing.T) {
	c := container.Container{ID: "abc123", Name: "web", State: "running", Host: "localhost"}
	line := time.Now().UTC().Format(time.RFC3339Nano) + ` <img src=x onerror="alert(1)">` + "\n"

	client := new(MockedClient)
	client.On("Host").Return(container.Host{ID: "localhost"})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{c}, nil)
	client.On("FindContainer", mock.Anything, "abc123").Return(c, nil)
	client.On("ContainerLogsBetweenDates", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(io.NopCloser(bytes.NewReader(makeMessage(line, container.STDOUT))), nil)

	manager := hostservice.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker.NewService(client, container.ContainerLabels{}))
	h := &handler{
		hostService: hostservice.NewMultiHostService(manager, 3*time.Second),
		config:      &Config{Base: "/", Authorization: Authorization{Provider: NONE}},
	}

	body := strings.NewReader(`{"containerExpression":"true","logExpression":"message contains \"onerror\""}`)
	req := httptest.NewRequest("POST", "/api/notifications/preview", body)
	rr := httptest.NewRecorder()

	h.previewExpression(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var result PreviewResult
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	require.Len(t, result.MatchedLogs, 1)
	assert.Equal(t, `&lt;img src=x onerror=&#34;alert(1)&#34;&gt;`, result.MatchedLogs[0].Message)
}
