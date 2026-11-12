package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFlags(t *testing.T, argv ...string) (Args, error) {
	t.Helper()
	var args Args
	p, err := arg.NewParser(arg.Config{}, &args)
	require.NoError(t, err)
	return args, p.Parse(argv)
}

func TestTypedFlags(t *testing.T) {
	args, err := parseFlags(t)
	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, args.Timeout)
	assert.Equal(t, time.Duration(0), args.AuthTTL.Duration(), "default session TTL")

	args, err = parseFlags(t, "--timeout", "3s", "--auth-ttl", "12h")
	require.NoError(t, err)
	assert.Equal(t, 3*time.Second, args.Timeout)
	assert.Equal(t, 12*time.Hour, args.AuthTTL.Duration())

	_, err = parseFlags(t, "--auth-ttl", "forever")
	assert.ErrorContains(t, err, "should be session or a duration like 12h")

	_, err = parseFlags(t, "--timeout", "soon")
	assert.Error(t, err)
}

func TestValidateServer(t *testing.T) {
	valid := func() Args {
		return Args{AuthProvider: "none", Mode: "server", ReleaseCheckMode: "automatic"}
	}

	tests := []struct {
		name    string
		modify  func(*Args)
		wantErr string
	}{
		{name: "defaults", modify: func(a *Args) {}},
		{name: "k8s mode", modify: func(a *Args) { a.Mode = "k8s" }},
		{name: "bad mode", modify: func(a *Args) { a.Mode = "cluster" }, wantErr: `invalid mode "cluster"`},
		{name: "bad provider", modify: func(a *Args) { a.AuthProvider = "ldap" }, wantErr: `invalid auth provider "ldap"`},
		{name: "bad release check", modify: func(a *Args) { a.ReleaseCheckMode = "never" }, wantErr: `invalid release check mode "never"`},
		{
			name:    "oidc missing client",
			modify:  func(a *Args) { a.AuthProvider = "oidc"; a.AuthOidcIssuer = "https://id.example.com" },
			wantErr: "are required with --auth-provider oidc",
		},
		{
			name: "oidc with github",
			modify: func(a *Args) {
				a.AuthProvider = "oidc"
				a.AuthOidcIssuer, a.AuthOidcClientID, a.AuthOidcClientSecret = "https://id.example.com", "id", "secret"
				a.AuthGithubClientID = "gh"
			},
			wantErr: "cannot be used with --auth-provider oidc",
		},
		{name: "short mcp token", modify: func(a *Args) { a.MCPToken = "short" }, wantErr: "at least 32 characters"},
		{name: "mcp token with space", modify: func(a *Args) { a.MCPToken = strings.Repeat("a", 32) + " b" }, wantErr: "must not contain whitespace"},
		{name: "good mcp token", modify: func(a *Args) { a.MCPToken = strings.Repeat("a", 32) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := valid()
			tt.modify(&args)
			err := args.validateServer()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateServerResolvesAliases(t *testing.T) {
	for _, alias := range []string{"github", "google"} {
		args := Args{AuthProvider: alias, Mode: "server", ReleaseCheckMode: "manual"}
		require.NoError(t, args.validateServer())
		assert.Equal(t, "simple", args.AuthProvider)
	}
}
