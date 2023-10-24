package web

import (
	"net/http"
)

func cspHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' gravatar.com data:; manifest-src 'self'; connect-src 'self' api.github.com;")
		next.ServeHTTP(w, r)
	})
}
