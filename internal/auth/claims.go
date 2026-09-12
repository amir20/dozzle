package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// claimSet is the claims a provider returned, in the order they should be
// searched: the ID token first, then userinfo. Keycloak puts resource_access in
// the ID token and only mirrors it into userinfo when the mapper says so, which
// is why the two are kept apart rather than merged into one map.
type claimSet []map[string]any

// claimPath is one dot-separated path, already split. It is a slice rather than
// a string so a client id with a dot in it can be a single segment.
type claimPath []string

func (p claimPath) String() string { return strings.Join(p, ".") }

// parseClaimPath splits an operator-supplied path such as
// "resource_access.dozzle.roles" on dots.
func parseClaimPath(path string) claimPath {
	var segments claimPath
	for segment := range strings.SplitSeq(strings.TrimSpace(path), ".") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}

	return segments
}

// lookup walks path through each claim source in turn and returns the first
// value it finds. A path that reaches a non-object before its last segment does
// not resolve in that source.
func (c claimSet) lookup(path claimPath) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	for _, source := range c {
		if value, ok := walk(source, path); ok {
			return value, true
		}
	}

	return nil, false
}

func walk(source map[string]any, path claimPath) (any, bool) {
	var current any = source
	for _, segment := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}

	return current, true
}

// strings resolves the first of paths that holds a list of strings. The list is
// returned even when it is empty: a claim that is present and empty is an answer
// ("this user has no roles"), and searching past it for a broader claim would
// turn that answer into a grant.
//
// Three shapes are accepted, because providers disagree: a JSON array of
// strings, a single comma or space separated string, and an object whose keys
// are the values, which is how Zitadel encodes project roles.
func (c claimSet) strings(paths []claimPath) ([]string, claimPath, bool) {
	for _, path := range paths {
		value, ok := c.lookup(path)
		if !ok {
			continue
		}

		if values, ok := claimStrings(value); ok {
			return values, path, true
		}
	}

	return nil, nil, false
}

func claimStrings(value any) ([]string, bool) {
	switch v := value.(type) {
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				values = append(values, strings.TrimSpace(s))
			}
		}
		return values, true
	case string:
		values := strings.FieldsFunc(v, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n'
		})
		return values, true
	case map[string]any:
		values := make([]string, 0, len(v))
		for key := range v {
			if strings.TrimSpace(key) != "" {
				values = append(values, strings.TrimSpace(key))
			}
		}
		return values, true
	default:
		return nil, false
	}
}

// decodeJWTClaims reads the payload of a compact JWS without checking its
// signature. See oidcProvider.identity for why the signature is not needed on
// an ID token that arrived straight from the token endpoint.
func decodeJWTClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("expected 3 segments, got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	return claims, nil
}
