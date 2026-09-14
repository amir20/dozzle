package cli

import (
	"testing"

	"github.com/amir20/dozzle/internal/config"
	"github.com/stretchr/testify/assert"
)

func lookupFrom(pairs map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := pairs[key]
		return v, ok
	}
}

func TestApplyConfigFile(t *testing.T) {
	fullFile := config.File{
		AuthProvider:  new("simple"),
		EnableActions: new(true),
		EnableShell:   new(true),
	}

	tests := []struct {
		name       string
		start      Args
		file       config.File
		argv       []string
		env        map[string]string
		wantAuth   string
		wantAct    bool
		wantShell  bool
		wantLocked Locked
	}{
		{
			name:     "empty file keeps defaults",
			start:    Args{AuthProvider: "none"},
			wantAuth: "none",
		},
		{
			name:      "file applies when nothing set",
			start:     Args{AuthProvider: "none"},
			file:      fullFile,
			wantAuth:  "simple",
			wantAct:   true,
			wantShell: true,
		},
		{
			name:       "flag locks auth provider",
			start:      Args{AuthProvider: "forward-proxy"},
			file:       fullFile,
			argv:       []string{"--auth-provider", "forward-proxy"},
			wantAuth:   "forward-proxy",
			wantAct:    true,
			wantShell:  true,
			wantLocked: Locked{AuthProvider: true},
		},
		{
			name:       "flag=value form locks",
			start:      Args{AuthProvider: "oidc"},
			file:       fullFile,
			argv:       []string{"--auth-provider=oidc", "--enable-actions=false"},
			wantAuth:   "oidc",
			wantShell:  true,
			wantLocked: Locked{AuthProvider: true, EnableActions: true},
		},
		{
			name:       "bare bool flag locks",
			start:      Args{AuthProvider: "none", EnableShell: true},
			file:       config.File{EnableShell: new(false)},
			argv:       []string{"--enable-shell"},
			wantAuth:   "none",
			wantShell:  true,
			wantLocked: Locked{EnableShell: true},
		},
		{
			name:       "env locks even when false or empty",
			start:      Args{AuthProvider: "none"},
			file:       fullFile,
			env:        map[string]string{"DOZZLE_ENABLE_ACTIONS": "false", "DOZZLE_ENABLE_SHELL": ""},
			wantAuth:   "simple",
			wantLocked: Locked{EnableActions: true, EnableShell: true},
		},
		{
			name:     "similar flag name and args after terminator do not lock",
			start:    Args{AuthProvider: "none"},
			file:     config.File{EnableActions: new(true)},
			argv:     []string{"--enable-actions-extra", "--", "--enable-actions"},
			wantAuth: "none",
			wantAct:  true,
		},
		{
			name:       "single and triple dash forms lock",
			start:      Args{AuthProvider: "none"},
			file:       fullFile,
			argv:       []string{"-auth-provider", "none", "-enable-actions=false", "---enable-shell"},
			wantAuth:   "none",
			wantShell:  false,
			wantLocked: Locked{AuthProvider: true, EnableActions: true, EnableShell: true},
		},
		{
			name:     "file false is applied",
			start:    Args{AuthProvider: "none", EnableActions: true},
			file:     config.File{EnableActions: new(false)},
			wantAuth: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.start
			applyConfigFile(&args, tt.file, tt.argv, lookupFrom(tt.env))
			assert.Equal(t, tt.wantAuth, args.AuthProvider)
			assert.Equal(t, tt.wantAct, args.EnableActions)
			assert.Equal(t, tt.wantShell, args.EnableShell)
			assert.Equal(t, tt.wantLocked, args.Locked)
		})
	}
}
