---
name: run-dev-instance
description: Use when Dozzle needs to be running in a browser to test a change by hand (clicking the UI, screenshots, driving it with Chrome or Playwright). Starts an instance on a free port derived from this worktree, so several worktrees can each have one at the same time, and never disturbs an instance someone else started.
---

# Run a Dozzle instance for testing

Several worktrees are usually checked out at once and more than one of them may already
be serving. **Never assume 3100 is yours** — it almost certainly belongs to another
worktree, and killing it kills someone else's session.

## Start one

```bash
make dev-auto   # backgrounded; see below
```

It prints the URL before anything else:

```
▸ dozzle dev on http://localhost:3356 (vite 5502, agent 7136)
```

The ports come from `scripts/dev-ports.mjs`, hashed from this checkout's path and then
walked forward until free, so **this worktree always gets the same URL** and two
worktrees never land on the same one. To know the URL without starting anything:

```bash
node scripts/dev-ports.mjs --json
```

Run it in the background and wait for the port rather than a fixed sleep:

```bash
PORT=$(node scripts/dev-ports.mjs --json | grep -o '"DOZZLE_PORT": [0-9]*' | grep -o '[0-9]*')
# start make dev-auto with run_in_background, then:
until curl -sf -o /dev/null http://localhost:$PORT/; do sleep 1; done
```

`make dev-auto` is `air` + `vite` with hot reload, which is what you want while iterating
on a change. For a one-shot check of already-written code, a production build is steadier
and has no vite half:

```bash
pnpm build && LIVE_FS=true go run . --level info --addr localhost:$PORT
```

## Overriding

`DOZZLE_PORT`, `VITE_PORT` and `AGENT_PORT` override any of the three, for `make dev`,
`make dev-auto`, `pnpm preview` and `pnpm agent:dev` alike. Plain `make dev` still uses
3100/5173/7007, which is what a human starting one by hand expects, and is deliberately
outside the auto-assigned ranges.

The Go server tells the dev page where Vite lives (`viteDevURL()` in
`internal/web/index.go`, read by `public/index.html`), so `VITE_PORT` is all that needs
setting — there is no hardcoded 5173 left.

## While testing

- Containers to look at: whatever is running locally (`docker ps`). No need to start any.
- Drive it with the Chrome extension (`mcp__claude-in-chrome__*`) when the user wants to
  watch, or with Playwright (`node_modules/playwright`, browsers already cached) for
  anything scripted or repeatable. Playwright is also the only way to fake a broken
  clipboard, a missing API, or an offline stream.
- Watch the console for errors the whole way through, not just at the end.

## Stop it

Kill only the process you started (the background task, or the PID on _your_ port):

```bash
lsof -tiTCP:$PORT -sTCP:LISTEN
```

Leave every other port alone. If a port you wanted is taken by another worktree, the
script has already moved past it — do not free it.
