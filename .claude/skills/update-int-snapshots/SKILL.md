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

2. **Run with `--update-snapshots` via `compose up`.** You must use `compose up` (not `compose run`) because the navigation sidebar snapshot includes the live container list, and the two modes produce different sibling containers. The cleanest way is to patch the playwright command in `docker-compose.yml` temporarily:

   ```bash
   sed -i.bak 's|command: npx --yes playwright test|command: npx --yes playwright test --update-snapshots|' docker-compose.yml
   make int
   mv docker-compose.yml.bak docker-compose.yml
   ```

   Snapshots must be generated in Linux/Chromium (the compose image), not macOS, because filenames include the platform suffix (e.g. `-chromium-linux.png`).

   Nothing needs clearing first. The compose file publishes no host ports, every container is named `dozzle_e2e_*` under the fixed project `dozzle_e2e`, and filters match those names exactly. Other containers on the machine can't block the run or show up in the snapshots, and a leftover run from another worktree is recreated. Two worktrees running `make int` at the same time do still clobber each other, so run one at a time.

3. **Verify** by rerunning the normal suite:

   ```bash
   make int
   ```

   Should now pass cleanly.

4. **Commit the regenerated PNGs** alongside the UI change so CI stays green.

## Common Mistakes

- **Running `npx playwright test --update-snapshots` locally on macOS.** Generates `-darwin` filenames CI doesn't use. Always go through the docker compose image.
- **Forgetting to delete first.** `--update-snapshots` does overwrite, but if the test layout changed (new test, renamed snapshot), stale PNGs are left behind. Wipe-and-regenerate is safest.
- **Adding a test container without the prefix, or a filter without anchors.** `name=dozzle` is a substring match and pulls in any local container with "dozzle" in its name. Use `container_name: dozzle_e2e_<service>` and `DOZZLE_FILTER=name=^dozzle_e2e_<service>$$` (`$$` escapes `$` in compose).
- **Publishing a host port "for debugging".** It brings back conflicts with whatever else is on the machine. Reach instances by service name from inside the network.

## Quick Reference

```bash
rm e2e/visual.spec.ts-snapshots/*.png
sed -i.bak 's|command: npx --yes playwright test|& --update-snapshots|' docker-compose.yml
make int
mv docker-compose.yml.bak docker-compose.yml
make int   # verify pass
```
