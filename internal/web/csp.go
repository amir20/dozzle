package web

import (
	"net/http"
)

func cspHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self' 'wasm-unsafe-eval' blob: https://cdn.jsdelivr.net https://*.duckdb.org; style-src 'self' 'unsafe-inline' blob:; img-src 'self' data:;",
		)
		next.ServeHTTP(w, r)
	})
}
