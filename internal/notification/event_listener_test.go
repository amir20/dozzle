package notification

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEvent(t *testing.T) {
	tests := []struct {
		name           string
		event          container.ContainerEvent
		wantName       string
		wantAttributes map[string]string
	}{
		{
			name:           "unhealthy health event is normalized",
			event:          container.ContainerEvent{Name: "health_status: unhealthy"},
			wantName:       "health_status",
			wantAttributes: map[string]string{"healthStatus": "unhealthy"},
		},
		{
			name:           "healthy health event is normalized",
			event:          container.ContainerEvent{Name: "health_status: healthy"},
			wantName:       "health_status",
			wantAttributes: map[string]string{"healthStatus": "healthy"},
		},
		{
			name: "existing attributes are preserved",
			event: container.ContainerEvent{
				Name:            "health_status: unhealthy",
				ActorAttributes: map[string]string{"name": "postgres"},
			},
			wantName:       "health_status",
			wantAttributes: map[string]string{"name": "postgres", "healthStatus": "unhealthy"},
		},
		{
			name:           "non-health events are left untouched",
			event:          container.ContainerEvent{Name: "die", ActorAttributes: map[string]string{"exitCode": "1"}},
			wantName:       "die",
			wantAttributes: map[string]string{"exitCode": "1"},
		},
		{
			name:           "bare health_status name is left untouched",
			event:          container.ContainerEvent{Name: "health_status"},
			wantName:       "health_status",
			wantAttributes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.event
			normalizeEvent(&event)
			assert.Equal(t, tt.wantName, event.Name)
			assert.Equal(t, tt.wantAttributes, event.ActorAttributes)
		})
	}
}

// Ensures the documented health alert expression actually matches a normalized event.
func TestNormalizedHealthEventMatchesSubscription(t *testing.T) {
	event := container.ContainerEvent{Name: "health_status: unhealthy"}
	normalizeEvent(&event)

	require.True(t, allowedEventNames[event.Name], "normalized event must pass the allowlist")

	sub := &Subscription{
		EventExpression: `name == "health_status" && attributes["healthStatus"] == "unhealthy"`,
	}
	program, err := expr.Compile(sub.EventExpression, expr.Env(types.NotificationEvent{}))
	require.NoError(t, err)
	sub.EventProgram = program

	notificationEvent := types.NotificationEvent{
		Name:       event.Name,
		Attributes: event.ActorAttributes,
	}
	assert.True(t, sub.MatchesEvent(notificationEvent))

	healthy := container.ContainerEvent{Name: "health_status: healthy"}
	normalizeEvent(&healthy)
	assert.False(t, sub.MatchesEvent(types.NotificationEvent{
		Name:       healthy.Name,
		Attributes: healthy.ActorAttributes,
	}))
}
