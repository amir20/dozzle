# Cloud Inside Dozzle

Design proposal for folding Dozzle Cloud's assistant, findings and alerts into the Dozzle UI, where
the log stream is already on screen and the app can act on what the answer says.

Status: proposal. Written against `bfcb2e1b` and doligence `main`, 2026-09-09.
Visual version: <https://claude.ai/code/artifact/b8610f69-dfef-4a9e-9ecd-2369d29f70f3>

## What already ships

A surprising amount of this is already wired. The gaps are findings, the assistant, and a place
where the day's activity is visible without leaving Dozzle.

| Already there                                                    | What it gives us                                                                                                                                    |
| ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `LogViewer/AlertLogItem.vue`, `LogViewer/CloudEventLogItem.vue`  | Cloud alerts splice into the log stream at the line that triggered them, anchored on `LogEvent.Id`.                                                 |
| `CloudHooks` (`internal/web/routes.go:74`)                       | The Dozzle → Cloud direction exists: `SearchLogs` and `GetAlerts` as unary gRPC calls proxied to `/api/cloud/*`. Every new read follows this shape. |
| `composable/cloudConfig.ts`, `common/ProBadge.vue`               | Plan and usage already reach the browser via `/api/cloud/status`, with an `isPro` computed for cosmetics.                                           |
| `common/SideDrawer.vue` + `createDrawer`, splitpanes pinned logs | Two shells for a chat surface, both already carrying real features.                                                                                 |
| _missing_                                                        | Findings. The daily / weekly scan output has no home in Dozzle at all.                                                                              |

## 1. Chat is a pane, not a widget

Cloud puts chat in a right-side slideover (`ui/app/components/AgentChatDrawer.vue`) because a browser
tab is all it has. Dozzle has something Cloud doesn't: the log stream is right there. Open chat as a
splitpane beside the logs, using the same mechanism that already powers pinned containers, so the
operator keeps watching the stream while asking about it. On mobile and narrow widths it degrades to
the existing side drawer.

### Every entry point carries context

The whole value is that the user never has to name what they are looking at. Each turn ships a small
`ViewContext`: route kind and ids, host, visible container names, the active search and level
filters, the visible time window from `scrollContext`, and any selected lines. Cloud folds it into
the per-turn context prompt it already builds for Telegram and Discord
(`api/internal/agent/agent.go:148`), so "why is this crashing" resolves without a single id typed.

- **⌘K palette** — a query that matches no container offers "Ask Dozzle: …" as the last row. This is
  Cloud's omnibox, on a palette Dozzle already has (`FuzzySearchModal.vue`).
- **Selected log lines** — "Ask about these lines" seeds the turn with the exact text, stack trace
  included.
- **Alert card** — "Why did this fire?" straight off `AlertLogItem`.
- **Container / host menus** — "Ask about this container".

### Chat that can act

The part Cloud structurally cannot match. An answer can offer to restart a container through the
local `containerActions` behind a confirm, and can link to `/container/{id}/time/{datetime}` — a real
destination inside the app the user is already in, not a link back out to a dashboard.

## 2. The round trip already exists

Add `Chat(ChatRequest) returns (stream ChatEvent)` to `CloudToolService` next to `SearchLogs` and
`GetAlerts`, authed with the same API key metadata. Dozzle proxies it to the browser as SSE. The
browser never touches cloud credentials, and the assistant gets live `rpc_*` tools back into this
same Dozzle for free, because Cloud already holds a `ToolStream` to it.

```
browser  --POST-->  dozzle  --Chat-->  cloud
        <--SSE---          <--events--
                              |
        existing ToolStream   |
        dozzle  <-------------+   rpc_list_containers, rpc_fetch_logs, ...
```

The turn leaves and re-enters the same process. That looks odd on a whiteboard and is already the
deployed topology: Cloud has never had another way to read a live container.

## 3. Findings land on the thing they describe

A finding is not an alert. It has a life: it opens, it ages while the problem persists, and it
resolves when the underlying pattern stops (`ui/app/composables/useFindings.ts`). Cloud lists them on
a page because a page is all it has. Dozzle knows which container each one is about, so the list is
the least interesting way to show them.

1. **On the container.** A severity dot on the container row in `ContainerTable.vue` and in
   `nav/NavItem.vue`. Click opens a drawer with headline, why, evidence and fix. Severity rides the
   dot, the row stays neutral, per the design system, and 20 containers stay scannable.
2. **Sidebar card.** `SideMenu.vue` is already a carousel of hosts, groups and services. Add an
   Activity card with open findings and recent alerts, count on the header. No new top-level nav.
3. **Since yesterday.** One line on `pages/index.vue` from the daily report Cloud already writes:
   `3 findings open · 2 alerts · 41 events kept quiet`, or simply "All clear".
4. **Events.** `GetAlerts` already returns matched events. Badge those log lines and add an "only
   alerted lines" chip to the filter row.

### The button Cloud can't have

Every finding carries the collapsed log patterns the scan actually read. In the Cloud UI that is a
link out. In Dozzle it is **"Show me the lines"** — drive the existing historical scroll to that
pattern's window and let the operator read the real stream. That single button is the argument for
putting findings here at all.

## 4. Instance scope is mostly already solved

Cloud derives `(user_id, api_key_id)` from the auth metadata and never reads it from a request body.
That is the whole requirement for alerts, search and findings: **the browser never sends an instance
id, and Dozzle never shows an instance picker.** Cloud's own `InstanceFilter` exists because that UI
serves many instances at once. This one serves one.

One real gap: the assistant's tools span every connected instance. Pin the turn by carrying this
instance's id in `ChatRequest` and telling the model the user is looking at it, and when an answer
does reach another instance, tag it visibly ("on homelab") rather than blending it into the same
prose.

## 5. Gate the work, not the app

Cloud already got the mechanism right and it transfers unchanged: `fixLocked` is computed server-side
from the current plan, never inferred from the data. A finding written before an upgrade keeps its
missing fix, and reading absence as tier is how paying customers got shown an upsell for what they
had already bought. Keep every plan decision in Cloud; Dozzle renders the gate it is handed and holds
no branching beyond `isPro` for cosmetics.

| Surface                    | Free                    | Pro              | Treatment when locked                                                                                            |
| -------------------------- | ----------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------- |
| Live logs, search, actions | full                    | full             | Never gated. Anything Dozzle does locally today stays free forever.                                              |
| Findings                   | headline, why, evidence | + prescribed fix | One muted line with a PRO pill. Not a card, not a gradient, not a modal.                                         |
| Assistant                  | smaller turn budget     | full budget      | The existing `rate_limit` code renders as "you're out of questions today" in an `InlineNotice`. Never a paywall. |
| Deep investigation         | on request              | on request       | Button visible and disabled with a reason, so the capability is discoverable without being sold.                 |
| History beyond retention   | plan window             | longer window    | The empty state states the window in days. That is the honest version of an upsell.                              |

**One upsell surface: `CloudPopover`.** Everything else stays silent. This falls out of the design
system's own rule that `primary` is reserved for the single action a surface wants you to take, and a
findings drawer wants you to press "Show me the lines", not "Upgrade".

## 6. Build order, starting with the chat pane

Chat first: highest visibility, and it reuses the most existing plumbing. Findings second, because
the drawer wants the log-jump behaviour the chat pane will already have proven.

1. **Proto and cloud service.** `Chat(ChatRequest) returns (stream ChatEvent)` on `CloudToolService`.
   `ChatRequest` carries the message, the session key and a `ViewContext` message. `ChatEvent`
   mirrors the four cases the web client already handles: status, delta, done, error.
   `protos/cloud.proto`, then `make generate`.
2. **Client method.** Add `Chat` beside `SearchLogs` in `internal/cloud/`, same dial and credentials
   path, streaming instead of unary. Surface it on `CloudHooks` so the web layer stays free of proto
   imports.
3. **SSE proxy.** `POST /api/cloud/chat` under the existing cloud-role route group in
   `internal/web/routes.go`, streaming events straight through. Cancel on client disconnect so a
   closed pane ends the cloud turn rather than paying for it.
4. **Cloud side: accept the channel.** A `dozzle:` channel in the agent's session resolution
   (`api/internal/agent/run.go:53`), plus rendering `ViewContext` into the per-turn context prompt.
   Tools, budget and fallbacks are untouched.
5. **The pane.** A chat pane in the splitpanes row, a composable that collects `ViewContext` from the
   current route and stores, and delta buffering on the same idiom as the log stream. Ship with one
   entry point (⌘K), then add the rest.
6. **Locales.** Every new key added to `en.yml` lands translated in all other `locales/*.yml` in the
   same commit. A key that exists only in English falls back silently and nothing flags it.

## Decide these before step 1

**Who owns the chat session?** Sessions are keyed by user and channel. A channel of
`dozzle:<api_key_id>` means every local Dozzle user shares one history. With simple auth and five
users that is probably wrong. Use `dozzle:<api_key_id>:<local_user>` unless the shared console is
deliberate.

**A Cloud instance is not a Dozzle host.** One Dozzle watches many Docker hosts. Findings carry a
host id, so map them against the local host list and drop findings for hosts this instance no longer
sees, rather than rendering a severity dot next to nothing.

**Does the pane survive navigation?** If chat stays open while the user moves from a container to a
host, the context changes under the conversation. Proposal: keep the pane, restate the context as a
small line in the thread when it changes, so the model and the reader see the same switch.
