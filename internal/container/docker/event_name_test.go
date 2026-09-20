package docker

import (
	"testing"

	"github.com/moby/moby/api/types/events"
	"github.com/stretchr/testify/assert"
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
