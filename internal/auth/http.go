package auth

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IsHTTPS reports whether the original client request used HTTPS, accounting
// for TLS terminated at an upstream reverse proxy via X-Forwarded-Proto.
func IsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// SetSessionCookie writes the session JWT. Password login and the OAuth callback
// both land here so a session is indistinguishable however it was proven. A zero
// ttl means a session cookie, which is what --auth-ttl=session asks for.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, token string, ttl time.Duration) {
	expires := time.Time{}
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   IsHTTPS(r),
		Expires:  expires,
	})
}

// ClearSessionCookie expires the session JWT.
func ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   IsHTTPS(r),
		Expires:  time.Unix(0, 0),
	})
}

// SafeRelativePath keeps a caller-supplied post-login redirect on this origin.
// "//evil.com" and "/\evil.com" are protocol-relative URLs that browsers follow
// off-site, so anything that is not a single-slash-rooted path is dropped rather
// than sanitized. Returns "" when the value is not safe to redirect to.
func SafeRelativePath(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") {
		return ""
	}

	if strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, `/\`) {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return ""
	}

	return raw
}
