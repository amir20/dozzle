package web

import (
	"fmt"
	"net/http"
)

// offlinePageTemplate is what the service worker shows when a navigation cannot
// reach the server. It is a Go template rather than a file under public/ for one
// reason: it needs the configured base to link home with, and a static file has
// no way to know it.
//
// Everything it needs is inline. It is shown precisely when the network is gone,
// so a stylesheet or a font it had to fetch would be the one thing it cannot
// have. No inline <script> either: the CSP allows 'unsafe-inline' for styles and
// not for scripts, so the retry is a plain link.
const offlinePageTemplate = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
    <meta name="robots" content="noindex" />
    <title>Offline</title>
    <style>
      :root {
        color-scheme: light dark;
        --bg: #f5f5f5;
        --fg: #1c1c1c;
        --muted: #1c1c1c99;
        --line: #1c1c1c26;
      }

      @media (prefers-color-scheme: dark) {
        :root {
          --bg: #121212;
          --fg: #e8e8e8;
          --muted: #e8e8e899;
          --line: #e8e8e826;
        }
      }

      body {
        margin: 0;
        min-height: 100dvh;
        display: grid;
        place-content: center;
        gap: 1rem;
        padding: 2rem calc(1rem + env(safe-area-inset-right)) calc(2rem + env(safe-area-inset-bottom))
          calc(1rem + env(safe-area-inset-left));
        background: var(--bg);
        color: var(--fg);
        text-align: center;
        font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
      }

      h1 {
        margin: 0;
        font-size: 1.5rem;
        font-weight: 700;
      }

      p {
        margin: 0;
        max-width: 32ch;
        color: var(--muted);
        font-size: 0.875rem;
        line-height: 1.5;
      }

      a {
        justify-self: center;
        padding: 0.625rem 1.25rem;
        border: 1px solid var(--line);
        border-radius: 0.5rem;
        color: inherit;
        font-size: 0.875rem;
        font-weight: 500;
        text-decoration: none;
      }
    </style>
  </head>
  <body>
    <h1>Dozzle is unreachable</h1>
    <p>This device is offline, or the server it streams logs from is not answering.</p>
    <a href="%s/">Try again</a>
  </body>
</html>
`

func (h *handler) offlinePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The service worker keeps its own copy per version, so the only job here is
	// to make sure an upgrade is not served a stale one on the way into that copy.
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprintf(w, offlinePageTemplate, basePrefix(h.config.Base))
}

// basePrefix is the configured base as a URL prefix, empty for the default "/"
// so paths built from it do not come out doubled.
func basePrefix(base string) string {
	if base == "/" {
		return ""
	}

	return base
}
