package web

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_cloudRoleGatesLinkingNotLooking pins the rule that the cloud role means
// "may link", not "may look". Reading cloud-backed data is open to any
// authenticated user and confined by their own filter; only linking and
// configuration are gated. Walking the tree rather than issuing requests keeps
// this honest if someone re-wraps the whole group in requireCloudRole again.
func Test_cloudRoleGatesLinkingNotLooking(t *testing.T) {
	h := restrictedHandler(t)
	// The walk only inspects which middlewares are attached, never runs them,
	// so an authorizer isn't needed — and createRouter refuses to build one
	// without it.
	h.config = &Config{Base: "/", Authorization: Authorization{Provider: NONE}}
	gate := reflect.ValueOf(h.requireCloudRole).Pointer()

	gated := map[string]bool{}
	require.NoError(t, chi.Walk(createRouter(h), func(method, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		for _, mw := range middlewares {
			if reflect.ValueOf(mw).Pointer() == gate {
				gated[method+" "+route] = true
				return nil
			}
		}
		gated[method+" "+route] = false
		return nil
	}))

	looking := []string{
		"GET /api/cloud/status",
		"GET /api/cloud/config",
		"GET /api/cloud/alerts",
		"GET /api/cloud/search/logs",
		"POST /api/cloud/feedback",
	}
	linking := []string{
		"PATCH /api/cloud/config",
		"DELETE /api/cloud/config",
		"GET /api/cloud/callback",
	}

	for _, r := range looking {
		locked, ok := gated[r]
		require.Truef(t, ok, "route %s not registered", r)
		assert.Falsef(t, locked, "%s is a read and must not require the cloud role", r)
	}
	for _, r := range linking {
		locked, ok := gated[r]
		require.Truef(t, ok, "route %s not registered", r)
		assert.Truef(t, locked, "%s changes the instance link and must require the cloud role", r)
	}
}
