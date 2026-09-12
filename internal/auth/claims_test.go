package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaimSetLookupSearchesSourcesInOrder(t *testing.T) {
	claims := claimSet{
		{"resource_access": map[string]any{"dozzle": map[string]any{"roles": []any{"shell"}}}},
		{"roles": []any{"actions"}, "resource_access": map[string]any{"dozzle": map[string]any{"roles": []any{"ignored"}}}},
	}

	value, ok := claims.lookup(claimPath{"resource_access", "dozzle", "roles"})
	require.True(t, ok)
	require.Equal(t, []any{"shell"}, value, "the ID token is searched before userinfo")

	value, ok = claims.lookup(claimPath{"roles"})
	require.True(t, ok)
	require.Equal(t, []any{"actions"}, value, "a claim only userinfo carries is still found")

	_, ok = claims.lookup(claimPath{"resource_access", "other", "roles"})
	require.False(t, ok)

	// Walking through a non-object does not resolve.
	_, ok = claims.lookup(claimPath{"roles", "nested"})
	require.False(t, ok)
}

func TestClaimStringsAcceptsEveryShape(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  []string
		ok    bool
	}{
		{"array", []any{"shell", " actions "}, []string{"shell", "actions"}, true},
		{"empty array", []any{}, []string{}, true},
		{"comma string", "shell,actions", []string{"shell", "actions"}, true},
		{"space string", "shell actions", []string{"shell", "actions"}, true},
		{"empty string", "", []string{}, true},
		{"object keys", map[string]any{"shell": map[string]any{"org": "example"}}, []string{"shell"}, true},
		{"number", 42.0, nil, false},
		{"bool", true, nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := claimStrings(tc.value)
			require.Equal(t, tc.ok, ok)
			if tc.ok {
				require.ElementsMatch(t, tc.want, got)
			}
		})
	}
}

// A claim that is present and empty is an answer, not a miss: the search must
// not fall through to a broader claim and turn "no roles" into a grant.
func TestClaimSetStringsStopsAtAnEmptyClaim(t *testing.T) {
	claims := claimSet{{"dozzle_roles": []any{}, "roles": []any{"shell"}}}

	values, path, ok := claims.strings([]claimPath{{"dozzle_roles"}, {"roles"}})
	require.True(t, ok)
	require.Empty(t, values)
	require.Equal(t, "dozzle_roles", path.String())
}

// A claim of the wrong shape is skipped, so a provider that puts a number at
// `roles` does not shadow a usable claim further down the list.
func TestClaimSetStringsSkipsUnusableShapes(t *testing.T) {
	claims := claimSet{{"dozzle_roles": 7.0, "roles": "shell"}}

	values, path, ok := claims.strings([]claimPath{{"dozzle_roles"}, {"roles"}})
	require.True(t, ok)
	require.Equal(t, []string{"shell"}, values)
	require.Equal(t, "roles", path.String())
}

func TestParseClaimPath(t *testing.T) {
	require.Equal(t, claimPath{"resource_access", "dozzle", "roles"}, parseClaimPath(" resource_access.dozzle.roles "))
	require.Equal(t, claimPath{"roles"}, parseClaimPath("roles"))
	require.Empty(t, parseClaimPath(""))
	require.Empty(t, parseClaimPath("."))
}

func TestDecodeJWTClaims(t *testing.T) {
	// {"sub":"abc","roles":["shell"]} with a bogus header and signature.
	claims, err := decodeJWTClaims("eyJhbGciOiJub25lIn0.eyJzdWIiOiJhYmMiLCJyb2xlcyI6WyJzaGVsbCJdfQ.sig")
	require.NoError(t, err)
	require.Equal(t, "abc", claims["sub"])
	require.Equal(t, []any{"shell"}, claims["roles"])

	_, err = decodeJWTClaims("not-a-jwt")
	require.Error(t, err)
}

func TestProfileKeyHashesUnsafeSubjects(t *testing.T) {
	require.Equal(t, "auth0|123", profileKey("auth0|123"))
	require.Equal(t, "3f2a9c1e-uuid", profileKey("3f2a9c1e-uuid"))

	hashed := profileKey("https://issuer.example.com/users/42")
	require.True(t, len(hashed) > len("oidc-"))
	require.NotContains(t, hashed, "/")
	require.Equal(t, hashed, profileKey("https://issuer.example.com/users/42"), "must be stable across logins")

	require.NotEqual(t, "..", profileKey(".."))
}
