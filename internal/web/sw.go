package web

import (
	"fmt"
	"net/http"
)

const serviceWorkerTemplate = `
const CACHE_NAME = "dozzle-%s";
const OFFLINE_URL = "%s/offline.html";

self.addEventListener("install", (event) => {
  // Fetched now so it is already there the first time the network is not. "reload"
  // bypasses the HTTP cache, so an upgrade cannot cache the previous version's copy.
  // A failure here must not reject: it would fail the install and leave the worker
  // unregistered, which is worse than having no fallback.
  event.waitUntil(
    caches
      .open(CACHE_NAME)
      .then((cache) => cache.add(new Request(OFFLINE_URL, { cache: "reload" })))
      .catch(() => {})
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    Promise.all([
      caches.keys().then((names) =>
        Promise.all(
          names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name))
        )
      ),
      // Navigations below go through this worker, and a worker that is not already
      // running has to boot before its fetch handler can ask the network for the
      // page. That boot sits in front of every cold navigation. Preload lets the
      // browser issue the request in parallel with the boot instead.
      self.registration.navigationPreload?.enable().catch(() => {}),
    ])
  );
  self.clients.claim();
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);

  // A navigation that cannot reach the server has nothing to fall back on, and a
  // home screen app has no browser chrome to say so: a launch on a phone whose
  // wifi has not reassociated yet is a blank screen with no error and no way to
  // retry. Still network-first, so nothing stale is ever shown while online.
  if (event.request.mode === "navigate") {
    event.respondWith(
      // The preloaded response is the same request, already in flight. It is
      // undefined when preload is unsupported or was not enabled in time, so the
      // plain fetch stays as the fallback.
      Promise.resolve(event.preloadResponse)
        .then((preloaded) => preloaded || fetch(event.request))
        .catch(() =>
          caches
            .open(CACHE_NAME)
            .then((cache) => cache.match(OFFLINE_URL))
            .then((cached) => cached || Response.error())
        )
    );
    return;
  }

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
	fmt.Fprintf(w, serviceWorkerTemplate, h.config.Version, basePrefix(h.config.Base))
}
