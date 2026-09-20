package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	docker "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDockerHealthEventName(t *testing.T) {
	tests := []struct {
		name   string
		action events.Action
		attrs  map[string]string
		want   string
	}{
		{
			name:   "docker healthy action is unchanged",
			action: "health_status: healthy",
			want:   "health_status: healthy",
		},
		{
			name:   "docker unhealthy action is unchanged",
			action: "health_status: unhealthy",
			want:   "health_status: unhealthy",
		},
		{
			name:   "start is unchanged",
			action: events.ActionStart,
			want:   "start",
		},
		{
			name:   "podman bare health_status uses attribute",
			action: "health_status",
			attrs:  map[string]string{"image": "nginx", "name": "web", "health_status": "healthy"},
			want:   "health_status: healthy",
		},
		{
			name:   "bare health_status without attribute stays bare",
			action: "health_status",
			want:   "health_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, dockerHealthEventName(tt.action, tt.attrs))
		})
	}
}

func TestResolveHealthEventNameInspectsBareAction(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "abc123def456").Return(
		docker.InspectResponse{
			ID: "abc123def456xxxx",
			State: &docker.State{
				Status: "running",
				Health: &docker.Health{Status: docker.Healthy},
			},
		}, nil,
	)
	d := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}
	name, attrs := d.resolveHealthEventName(context.Background(), "health_status", map[string]string{"image": "nginx"}, "abc123def456")
	assert.Equal(t, "health_status: healthy", name)
	assert.Equal(t, "healthy", attrs["healthStatus"])
	assert.Equal(t, "nginx", attrs["image"])
	proxy.AssertExpectations(t)
}

func TestResolveHealthEventNameInspectsStarting(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "abc123def456").Return(
		docker.InspectResponse{
			ID: "abc123def456xxxx",
			State: &docker.State{
				Status: "running",
				Health: &docker.Health{Status: docker.Starting},
			},
		}, nil,
	)
	d := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}
	name, attrs := d.resolveHealthEventName(context.Background(), "health_status", nil, "abc123def456")
	assert.Equal(t, "health_status: starting", name)
	assert.Equal(t, "starting", attrs["healthStatus"])
	proxy.AssertExpectations(t)
}

func TestResolveHealthEventNameSkipsInspectWhenDockerEncodesStatus(t *testing.T) {
	proxy := new(mockedProxy)
	d := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}
	name, attrs := d.resolveHealthEventName(context.Background(), "health_status: unhealthy", map[string]string{"name": "web"}, "abc123def456")
	assert.Equal(t, "health_status: unhealthy", name)
	assert.Equal(t, "web", attrs["name"])
	proxy.AssertNotCalled(t, "ContainerInspect", mock.Anything, mock.Anything)
}

func TestResolveHealthEventNameInspectErrorKeepsBareAction(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerInspect", mock.Anything, "abc123def456").Return(docker.InspectResponse{}, errors.New("not found"))
	d := &Client{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}
	name, attrs := d.resolveHealthEventName(context.Background(), "health_status", map[string]string{"image": "nginx"}, "abc123def456")
	assert.Equal(t, "health_status", name)
	assert.Equal(t, "nginx", attrs["image"])
	proxy.AssertExpectations(t)
}
