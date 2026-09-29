package web

import (
	"net/http"
)

// DuckDB's wasm, worker and extensions are all served by Dozzle, so nothing is loaded
// from a third party. blob: covers the worker's importScripts shim, and
// 'wasm-unsafe-eval' lets it compile the wasm.
const contentSecurityPolicy = "default-src 'self' 'wasm-unsafe-eval' blob:; " +
	"script-src 'self' 'wasm-unsafe-eval' blob:; " +
	"style-src 'self' 'unsafe-inline' blob:; img-src 'self' data:; font-src 'self' data:; " +
	"object-src 'none'; base-uri 'self';"

func cspHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}
