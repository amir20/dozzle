# Search progress / completion indicator

Status: built. Backend emits `search-status`, frontend gates the false "No logs" and renders a slim status bar.

Two refinements added during implementation, beyond the original design:

- The in-progress bar reveals only after a 400ms delay, so fast searches (the common case) never flash it. Only slow searches surface it.
- The completion summary (capped/exhausted) shows only for searches that actually ran slow; fast searches stay quiet. Zero-match searches always show "No matches · searched all logs" (the real fix for the false "No logs").

## Problem (issue #4769)

Regex search over a large log with sparse matches takes 10-15s. During that time the UI gives no signal that the search is still running, and can even render "No logs" mid-search, so the user can't tell whether it's done.

Reporter's three asks:

1. Search progress indication. The spinner exists but does not track the search.
2. Faster partial results. Already streams per-window; not the real gap.
3. "How much of the log was searched." Genuinely missing.

## Root cause / mechanics

`internal/web/logs.go:390-439`, the backfill goroutine:

- Walks backward in expanding time windows. `delta` starts at `-10s` and doubles each iteration (`:392`, `:430`).
- Loops until 50 matches accumulate (`minimum`) or `!stillRunning` (`:426`, reached the oldest log / no container still running).
- Filtering is server-side Go regex (`matchesFilter` `:44-50`, applied `:416`). The Docker API only does time-range fetch (`since`/`until`); it has no text search, so Dozzle does the windowing and matching itself.
- Per-window matches stream out via the `backfill` channel -> `logs-backfill` SSE (`:435-437`, emitted `:501-507`) -> frontend prepends them (`assets/composable/eventStreams.ts:180-184`).
- No completion signal is ever sent. The goroutine just `return`s at both exits.

The "No logs" bug: `loading` flips to false on `onopen` (`eventStreams.ts:196`), which fires the instant the SSE connection opens, before backfill does any work. The skeleton only persists 3s while empty (`EventSource.vue:2`, `:37-38`); after that, if no match has arrived yet, the UI shows "No logs" (`EventSource.vue:9`) while the search is still running.

Everything the three asks need (`to` boundary, match count, which exit fired) is already computed in the goroutine and thrown away. The fix is to surface it.

## Design: one new SSE event `search-status`

```jsonc
// each backfill iteration, plus once more at the end with done=true
{ "scannedTo": "2026-06-01T14:31:00Z", "matches": 12, "done": false }
{ "scannedTo": "2026-06-01T13:10:00Z", "matches": 50, "done": true, "reason": "capped" }    // hit 50 cap, more may exist further back
{ "scannedTo": "2026-04-02T00:00:00Z", "matches": 23, "done": true, "reason": "exhausted" } // ran out of logs, nothing older
```

`reason` only set when `done`.

## Changes

### 1. Backend `internal/web/logs.go` (~20 lines, no search-logic change)

- Add a `searchStatus` struct and a `searchStatusCh` channel.
- In the goroutine: track `found`; emit a progress event after `to = to.Add(delta)`; emit a terminal event at both exits (`exhausted` at `:426`, `capped` after the loop ends).
- Use a ctx-guarded send to avoid blocking forever if the main loop already exited:
  ```go
  send := func(s searchStatus) {
      select {
      case searchStatusCh <- s:
      case <-r.Context().Done():
      }
  }
  ```
  (The existing `backfill <-` send has the same latent leak; worth fixing in passing.)
- In the select loop (`:479-515`) add: `case s := <-searchStatusCh: sseWriter.Event("search-status", s)`.

### 2. Frontend `assets/composable/eventStreams.ts`

- Add a `searchStatus` ref `{ active, done, matches, scannedTo?, reason? }`.
- Reset it in `connect()` with `active: isSearching.value` so State 1 shows immediately, before the first event.
- `es.addEventListener("search-status", ...)` to populate it.
- Return `searchStatus` from `useLogStream`.

### 3. Frontend `assets/components/LogViewer/EventSource.vue`

- Pull `searchStatus` from the stream source.
- Render `<SearchStatus :status="searchStatus" class="sticky top-0 z-10" />` when `searchStatus.active`.
- The actual bug fix: gate the empty state with `&& !searchStatus.active` so "No logs" never shows during a search.

### 4. New `assets/components/LogViewer/SearchStatus.vue` (~40 lines)

Slim sticky bar, states: searching / capped / exhausted / empty. Reuse the `IndeterminateBar` idiom while active. `tabular-nums` on the count so it doesn't jitter. Primary check on done. Restrained styling, no banner or card.

"Load older" is not new wiring: the existing `LoadMoreLogEntry` (`eventStreams.ts:127`, `assets/composable/logLoader.ts`) already re-runs the fetch with the active filter. On `capped`, the copy just points at it.

### 5. i18n `locales/en.yml` under `label:` after `:61`

```yaml
search-status:
  searching: Searching older logs…
  searching-to: "Searching older logs… back to {time}"
  matches: "no matches | {count} match | {count} matches"
  capped: "{count} matches · searched back to {time}"
  exhausted: "Searched all logs · {count} matches"
  empty: No matches · searched all logs
```

## Decisions to lock before building

1. Show the status for any filtered backfill (regex + level filters) or regex-only? Lean both: it's free and self-gates, since the goroutine only runs for filtered views (`:387`).
2. Optional: turn the bottom `IndeterminateBar` into a real fill bar, `(now - to) / (now - container.Created)`. Pure frontend, single-container only (multi-container has no single floor). Defer unless wanted.
3. Placement: sticky top of the scroll area (proposed) vs inline first row. Lean sticky so it stays visible while scrolling.

## Scope

~20 lines backend, ~15 lines across two existing frontend files, one ~40-line component, six i18n strings. No change to how search works.
