package web

import (
	"fmt"
	"net/http"
)

const serviceWorkerTemplate = `
const CACHE_NAME = "dozzle-%s";

self.addEventListener("install", (event) => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((names) =>
      Promise.all(
        names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name))
      )
    )
  );
  self.clients.claim();
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);

  // Cache immutable hashed assets. Rolldown appends a base64url hash after a dash
  // (main-DDlQ-1D9.js), not a dot-separated hex one, so match that shape.
  if (url.pathname.match(/\/assets\/.+-[A-Za-z0-9_-]{8,}\.[a-z0-9]+$/)) {
    event.respondWith(
      caches.match(event.request).then((cached) => {
        if (cached) return cached;
        return fetch(event.request).then((response) => {
          if (response.ok) {
            const clone = response.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(event.request, clone));
          }
          return response;
        });
      })
    );
    return;
  }

  // Network-first for everything else (API calls, HTML, etc.)
});
`

func (h *handler) serviceWorker(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprintf(w, serviceWorkerTemplate, h.config.Version)
}
