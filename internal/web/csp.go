package web

import (
	"net/http"
)

// script-src is spelled out so it does not fall back to default-src, which has to
// allow all of cdn.jsdelivr.net for duckdb's wasm fetches. Scripts are limited to
// the @duckdb npm scope, which only DuckDB can publish to; the worker loads its
// bundle from there with importScripts.
const contentSecurityPolicy = "default-src 'self' 'wasm-unsafe-eval' blob: https://cdn.jsdelivr.net https://*.duckdb.org; " +
	"script-src 'self' 'wasm-unsafe-eval' blob: https://cdn.jsdelivr.net/npm/@duckdb/; " +
	"style-src 'self' 'unsafe-inline' blob:; img-src 'self' data:; font-src 'self' data:; " +
	"object-src 'none'; base-uri 'self';"

func cspHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}
