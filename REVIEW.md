# REVIEW.md

What a review of this repo looks for, and how it reports. Read by the CI reviewer
(`.github/workflows/claude-code-review.yml`), by local `/code-review`, and by the
fresh-context review step in `/verify`. `CLAUDE.md` holds the conventions; this file
says which breakages matter most and how to report them.

Dozzle is widely deployed next to the Docker socket, often on a network. A bug here
runs on thousands of hosts nobody here can patch, until each one pulls a new image.

## What failure looks like here

Each of these shipped at least once and was fixed later. Treat any of them as **Important**.

- **A remote user reaches the Docker API, or another user's data.** Check auth or role
  bypass, missing escaping (XSS in log lines, container names, labels), path traversal,
  and SSRF through any user-supplied URL. Secrets in logs count too. Check all three
  auth modes (none, simple, forward-proxy), not only the one the PR was tested in.
- **A gate on one side only.** The server refuses but the UI still offers the action
  (the self-update button shown to users the server answers 403). Or one path checks
  a label or role that another path skips (`/update/self` ignored
  `dev.dozzle.update=check`). When the diff gates something, find every other way to
  reach it: the HTTP route, the cloud tool, MCP, the agent RPC.
- **A read with no bound.** Anything read from a container, the daemon or a client
  needs a limit: a log line with no newline (a TTY progress bar) grew memory with the
  size of the log, and MCP log tools read a whole time range. Reading must also stop
  once the client is gone.
- **Shared work on one caller's context, or a cached failure.** A lookup shared across
  callers ran on the first caller's ctx, and its timeout was cached for 5 minutes for
  every container on that digest. Shared work runs detached with its own timeout, and
  errors and timeouts are not cached.
- **State marked before the action it records.** `update_available` was marked raised
  before the send, so one dropped on a full channel never came back. Mark after it
  succeeds.
- **Data that escapes its lock.** Returning a backing slice or map that a later write
  can mutate (the ring buffer's snapshot, #5526). Copy under the lock. A diff that touches
  shared state needs `go test -race` to pass on a test that exercises it.
- **One request that takes the process down.** `log.Fatal`, `panic` or an unchecked
  index in a handler (`/api/hosts//...` killed the server, #5453). A bad request fails
  that request only.
- **State missed while a stream was down.** Anything built from Docker events must
  survive the event stream dropping and reconnecting: re-list, then reconcile
  (containers started during the gap had no stats, #5429).
- **Registered in one list but not its twin.** A usage key counted but missing from
  `UsageKeys`. A key in `locales/en.yml` but not in the other locales. An English docs
  page changed without its `zh`/`de`/`fr`/`es` mirrors (CLAUDE.md, Translations). A
  proto field added on one side of the agent RPC.
- **An empty filter that means "everything".** Copy with every level cleared returned
  every level. Empty means none, unless the code says otherwise on purpose.
- **A bulk action wider than its confirmation.** A bulk action must touch only the
  containers its confirm listed.
- **A default that changes for existing users.** A changed default, renamed flag or env
  var, or moved config path changes behaviour on every host that upgrades. Call it out
  even when the new behaviour is better.
- **Backend package boundaries.** `internal/container` importing a platform package,
  or one platform importing another (CLAUDE.md, Where backend code goes).
- **Gating inferred from absence.** Plan decisions come from Cloud already decided.
  Branching on missing data to pick a tier is a bug (CLAUDE.md, Gating).

## Evidence

- Every finding names a `file:line` and quotes or describes the code there. Don't
  guess what a function does from its name: open it.
- For a cross-file finding (a gate's other path, a list's twin), name both locations.
- If something can't be confirmed from the code, leave it out. Don't post it as a question.

## Tags

- **Important**: wrong behaviour, a broken rule above, a security hole, a failing build.
  Must be fixed or answered before merge.
- **Nit**: works, but is clearer or simpler another way. **At most three per review**;
  drop the rest.
- **Pre-existing**: a real bug in code this PR didn't change. Say so, so it isn't
  blamed on the PR. Don't ask the PR to fix it.

Not findings: style, renames, speculative abstractions, extra validation against
trusted internal code, tests for trivial code.

## Dependency updates

A version bump's diff is a line in `package.json` or `go.mod` plus lockfiles, which
this review skips. So the review is of the _changes upstream_:

- Read the release notes in the PR body (`gh pr view`). Any 0.x minor counts as a
  major: `0.5 → 0.6` is allowed to break. When a note only links the change ("⚠️
  move to X (#546)"), open the linked PR or the project's migration guide with
  WebFetch: that's where the breaking details are.
- For each breaking change, deprecation or behaviour change, grep this repo for the
  affected API, option, env var or type, and name each use as a finding with its
  `file:line`: what it does today and what the new version does instead.
- A change CI can't see is **Important**: a renamed option or env var, a changed
  default, a runtime-only failure, anything that changes what existing users get.
  A type error CI will report anyway is not a finding.
- A security release says so in the summary, so it gets merged first.
- If the notes say nothing that touches this repo's usage, say so: "0 findings.
  Release notes checked for X, Y; no affected usage."

## Skip

- Generated code: `*.pb.go`, `assets/types/generated/`
- `bun.lock`, `go.sum`
- `locales/*.yml` other than `en.yml`, and translated docs (`docs/{zh,de,fr,es}/`),
  except to check that a key or page changed in English changed in all of them
- `assets/icons/` and other image assets

## Summary format

Start with a count, then list the findings:

> **2 Important, 1 Nit, 0 Pre-existing.**

When there is nothing: **"0 findings. Nothing to fix."** Then say what was read (the
whole diff, or which parts) and why the change is correct, citing `file:line` for
what was verified. Don't pad a clean review with praise or a restatement of the PR.
