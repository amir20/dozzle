package web

import (
	"net/http"
	"path"
	"strings"
)

// Built assets are brotli-compressed once at build time by scripts/compress-dist.js,
// which replaces each original with a `.br` sibling. These are the types it stores
// precompressed; everything else is served as-is.
var compressedTypes = map[string]string{
	".js":   "text/javascript; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".svg":  "image/svg+xml",
	".json": "application/json",
	".map":  "application/json",
}

func acceptsBrotli(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "br")
}

// contentTypeFor resolves the type from the original name, since the file on disk is
// the `.br` sibling. The table is hardcoded rather than read via mime.TypeByExtension
// because the runtime image has no system mime database.
func contentTypeFor(name string) string {
	if t, ok := compressedTypes[path.Ext(name)]; ok {
		return t
	}
	return "application/octet-stream"
}
