package cli

import (
	"testing"

	"github.com/amir20/dozzle/internal/container/agent"
	"github.com/stretchr/testify/assert"
)

// An agent reads the hub's own flags: unset leaves it to the hub, false refuses,
// and true is the same as unset.
func TestAgentRestrictions(t *testing.T) {
	tests := []struct {
		name string
		args Args
		argv []string
		env  map[string]string
		want agent.Restrictions
	}{
		{name: "unset leaves it to the hub", argv: []string{"agent"}},
		{
			name: "env false refuses",
			argv: []string{"agent"},
			env:  map[string]string{"DOZZLE_ENABLE_ACTIONS": "false", "DOZZLE_ENABLE_SHELL": "false"},
			want: agent.Restrictions{DisableActions: true, DisableShell: true},
		},
		{
			name: "flag false refuses only that one",
			argv: []string{"--enable-shell=false", "agent"},
			want: agent.Restrictions{DisableShell: true},
		},
		{
			name: "true is the same as unset",
			args: Args{EnableActions: true, EnableShell: true},
			argv: []string{"--enable-actions", "agent"},
			env:  map[string]string{"DOZZLE_ENABLE_SHELL": "true"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, agentRestrictions(tt.args, tt.argv, lookupFrom(tt.env)))
		})
	}
}
