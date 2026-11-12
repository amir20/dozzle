---
name: verify
description: Prove a change works before opening a PR for Dozzle. Runs the checks CI runs for what the diff touches, shows the new behaviour actually working, and has a fresh-context subagent review the diff against REVIEW.md, then writes the PR's Tests section from the real output. Use before every `gh pr create`, and whenever the user says "verify", "check it works", "is this ready for a PR" or "/verify".
---

# Verify before the PR

A PR is opened only after this has passed. "It should work" or "the tests pass" is not
evidence: every claim in the PR's Tests section is a command that was run and its real
output. If a step can't run (no Docker, no Chrome), say so in the Tests section. Don't
skip it quietly.

Work on the feature branch with the change committed, so the review step sees one diff.
Run `git fetch origin` first and compare against `origin/main`, which is what the CI
reviewer diffs against. A worktree's local `main` is often behind, and then every step
below pulls in commits that already merged: `git diff origin/main...HEAD`.

## 1. Checks for what the diff touches

List the changed paths (`git diff --name-only origin/main...HEAD`) and run the matching blocks.
These are the commands CI's `_tests.yml` runs, so a pass here predicts a pass there.

**Go (`*.go`, `go.mod`, `protos/`, `proto/`)**

```bash
just test                      # go test -race -count 1, after go generate
go tool staticcheck ./...
go tool tygo generate && git diff --exit-code assets/types/generated
! grep -n 'any /\* ' assets/types/generated/*.ts   # an unmapped type; CI fails on it
```

If `just test` regenerated protobuf code, commit it: CI runs `just test-ci`, which
doesn't regenerate.

**Frontend (`assets/`, `*.vue`, `*.ts`)**

```bash
bun run typecheck && bun run test && bun run lint
```

**Translations (`locales/`, `docs/`)**

```bash
bun docs/scripts/check-translations.mjs
```

A new or renamed key in `locales/en.yml` must be in every other locale, translated
(CLAUDE.md, Translations). A changed English docs page needs its four mirrors.

**Helm chart (`charts/`)**

```bash
helm lint charts/dozzle --strict
helm template dozzle charts/dozzle --set rbac.actions=true,rbac.nodeStats=true,ingress.enabled=true,persistence.existingClaim=data >/dev/null
```

Fix every failure before moving on. Don't call a failure "pre-existing" without
showing it also fails on `main`.

## 2. Show the change working

Passing the suite isn't the same as the change working. Prove the specific behaviour
the PR claims, with the narrowest check that exercises it:

- **Bug fix:** a test that reproduces the bug. Show it **fails on main** and passes
  with the fix, and say so in the PR ("fails on main"). Do it in a scratch worktree so
  the branch is never touched:
  ```bash
  wt=$(mktemp -d) && git worktree add --detach "$wt" HEAD
  (cd "$wt" && git checkout origin/main -- <the non-test files the fix changed> && go test -race ./internal/<pkg> -run <TestName>)
  git worktree remove --force "$wt"
  ```
  The run must fail. If it passes, the test doesn't cover the bug: rewrite it.
- **Concurrency or shared state:** the test runs under `-race` and drives the
  concurrent path (as in #5526), not just the single-threaded one.
- **New Go behaviour:** run the covering tests by name with `-run` and `-v`, and quote
  the lines that show the asserted values. If no test covers it, write one.
- **UI change:** start an instance with the `run-dev-instance` skill and screenshot
  the changed screen with Playwright, in light and dark mode if styling changed, and
  at phone width if the layout changed. Compare it with the closest existing screen
  (CLAUDE.md, Design System). If visual snapshots changed, use `update-int-snapshots`.
- **Agent, swarm or k8s path:** say which mode was exercised. A change tested in
  server mode only is untested in the others; name them under "Not run".
- **Auth or roles:** exercise the change in every auth mode it touches (none, simple,
  oidc, forward-proxy), and as a user without the role.

## 3. Fresh-context review

Launch one `general-purpose` subagent. Give it only this brief, not your own reasoning
or a summary of what you think the change does. A reviewer handed your conclusion
tends to confirm it.

> Review the diff `git diff origin/main...HEAD` in <this worktree's path>. Read `REVIEW.md`
> first and follow it exactly: what failure looks like here, the file:line evidence
> rule, the Important / Nit / Pre-existing tags, the skip list and the summary format.
> Read the code around the diff: callers, and every other path to anything the diff
> gates, bounds or registers. Don't edit anything. Return the findings in REVIEW.md's
> format.

For each **Important** finding: fix it, or write one line on why it's wrong. Rerun step
1 for what you touched after a fix. Don't add new mechanisms while fixing; a fix that
needs one is a scope question for the user (CLAUDE.md, Scope Discipline). Run the
review once, plus once more if a fix changed behaviour. Never loop it.

## 4. Write the Tests section

Put this in the PR description, filled from the real runs above:

```markdown
### Tests

- Go: just test (-race), staticcheck, tygo diff: pass
- Frontend: typecheck, vitest, lint: pass
- Behaviour: <the test or screenshot, and what it showed: "TestX fails on main
  (got [4 5 6], want [1 2 3]), passes here">
- Fresh-context review: <count line>; <each Important and what was done>
- Not run: <anything skipped, and why: other deployment modes, Docker-only e2e>
```

Keep it to what was run. A line that can't point at a command doesn't go in.
