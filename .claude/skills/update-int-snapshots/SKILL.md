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

3. **Run with `--update-snapshots` via `compose up`.** You must use `compose up` (not `compose run`) because the navigation sidebar snapshot includes the live container list, and the two modes produce different sibling containers. The cleanest way is to patch the playwright command in `docker-compose.yml` temporarily:

   ```bash
   sed -i.bak 's|command: npx --yes playwright test|command: npx --yes playwright test --update-snapshots|' docker-compose.yml
   make int
   mv docker-compose.yml.bak docker-compose.yml
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

```bash
rm e2e/visual.spec.ts-snapshots/*.png
docker ps | grep ':8080->'   # confirm port free
sed -i.bak 's|command: npx --yes playwright test|& --update-snapshots|' docker-compose.yml
make int
mv docker-compose.yml.bak docker-compose.yml
make int   # verify pass
```
