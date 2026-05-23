---
name: update-int-snapshots
description: Use when Playwright visual snapshots in e2e/visual.spec.ts-snapshots/ need to be regenerated after a UI change (logo, theme, layout, anything that affects rendered pixels) and the integration suite needs to be rerun via make int
---

# Update Integration Snapshots

## When to Use

- A visual change landed (logo, color, spacing, fonts) and `make int` will fail on `visual.spec.ts` until snapshots are regenerated.
- A snapshot test reports diffs you've confirmed are intentional.

Do NOT use when diffs are unintentional regressions — investigate first.

## Procedure

1. **Delete the stale snapshots** so Playwright writes fresh ones (don't try to update in place, the old PNGs can confuse the diff):

   ```bash
   rm e2e/visual.spec.ts-snapshots/*.png
   ```

2. **Check for port 8080 conflicts.** The `custom_base` test container binds host port 8080. If another container is already on it (common: `doligence-api-1`), the run fails with `Bind for 0.0.0.0:8080 failed: port is already allocated`.

   ```bash
   docker ps --format '{{.Names}}\t{{.Ports}}' | grep ':8080->'
   ```

   If anything other than test containers is bound, stop it first (ask the user before stopping containers from other projects).

3. **Run Playwright with `--update-snapshots`.** Bypass the Makefile so we can pass the flag:

   ```bash
   docker compose run --rm --build playwright npx playwright test --update-snapshots
   ```

   Snapshots must be generated in Linux/Chromium (the compose image), not macOS, because filenames include the platform suffix (e.g. `-chromium-linux.png`).

4. **Verify** by rerunning the normal suite:

   ```bash
   make int
   ```

   Should now pass cleanly.

5. **Commit the regenerated PNGs** alongside the UI change so CI stays green.

## Common Mistakes

- **Running `npx playwright test --update-snapshots` locally on macOS.** Generates `-darwin` filenames CI doesn't use. Always go through the docker compose image.
- **Forgetting to delete first.** `--update-snapshots` does overwrite, but if the test layout changed (new test, renamed snapshot), stale PNGs are left behind. Wipe-and-regenerate is safest.
- **Stopping unrelated containers without asking.** The port conflict is annoying but the user's other dev containers may be holding important state.

## Quick Reference

| Step            | Command                                                                             |
| --------------- | ----------------------------------------------------------------------------------- |
| Wipe snapshots  | `rm e2e/visual.spec.ts-snapshots/*.png`                                             |
| Check port 8080 | `docker ps \| grep ':8080->'`                                                       |
| Regenerate      | `docker compose run --rm --build playwright npx playwright test --update-snapshots` |
| Verify          | `make int`                                                                          |
