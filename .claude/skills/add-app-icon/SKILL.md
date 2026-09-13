---
name: add-app-icon
description: Use when adding a bundled app icon to Dozzle for a container image that shows no logo, or when an image resolves to the wrong icon
---

# Add App Icon

Icons live in `assets/icons/apps/<slug>.svg` (or `.webp`). The filename without the
extension is the slug. `assets/utils/appIcons.ts` resolves an image reference to a slug.

## Procedure

1. Find how the image resolves today. The resolver drops registry, tag and digest, then
   tries the image name and falls back to the namespace (`linuxserver/sonarr` tries
   `sonarr`). Distributor namespaces (`DISTRIBUTORS`) and generic names (`GENERIC`, e.g.
   `server`, `app`) are skipped. Suffixes like `-server`, `-app`, `-alpine` are stripped.

2. Get the icon from dashboard-icons. Prefer SVG, use WebP only when no SVG exists.
   Never use PNG (it is not globbed). Check the upstream tree rather than guessing URLs,
   since variants can exist in one format and not another:

   ```bash
   gh api 'repos/homarr-labs/dashboard-icons/git/trees/main?recursive=1' \
     --jq '.tree[].path' | grep -E '^(svg|webp)/<slug>(-light|-dark)?\.'
   ```

   ```bash
   slug=<slug>
   curl -fsSL -o assets/icons/apps/$slug.svg \
     https://raw.githubusercontent.com/homarr-labs/dashboard-icons/main/svg/$slug.svg
   ```

   Upstream WebPs are 128-4096px and often 20-180KB. Downscale to 64px tall, lossy, like
   every other WebP in the folder (they land around 1-3KB):

   ```bash
   curl -fsSL -o /tmp/$slug.webp \
     https://raw.githubusercontent.com/homarr-labs/dashboard-icons/main/webp/$slug.webp
   cwebp -quiet -resize 0 64 -q 80 -m 6 /tmp/$slug.webp -o assets/icons/apps/$slug.webp
   ```

   Look at the result before committing. Do not substitute the app's own `favicon.svg`
   when upstream has no SVG: those are often a PNG wrapped in an `<image>` tag, not
   vector artwork.

   If upstream has themed variants, add them too: `<slug>-light.svg` is artwork for dark
   backgrounds, `<slug>-dark.svg` is for light backgrounds. The base `<slug>.svg` is still
   required, since it is what the resolver checks for.

   Use the upstream slug as the filename. Do not rename it to match the image.

3. If the image name does not match the slug after step 1, add an entry to `ALIASES` in
   `assets/utils/appIcons.ts`, keyed by the lowercased image name or namespace:

   ```ts
   "signal-cli-rest-api": "signal",
   ```

   Do not alias a generic word or a distributor namespace. If a new distributor
   repackages other people's software, add it to `DISTRIBUTORS` instead.

4. Add a case to the `iconSlugForImage` `test.each` table in
   `assets/utils/appIcons.spec.ts` for every alias you added:

   ```ts
   ["bbernhard/signal-cli-rest-api:latest", "signal"],
   ```

5. Verify:

   ```bash
   TZ=UTC pnpm test assets/utils/appIcons.spec.ts
   pnpm exec prettier --write assets/utils/appIcons.ts assets/utils/appIcons.spec.ts
   ```

## Rules

- One commit per batch, titled `feat(icons): add icons for <apps>`.
- Only well-known, publicly available images. No logos for private or internal images;
  users can already pick an existing icon with the `dev.dozzle.icon=<slug>` label, or
  hide a wrong guess with `dev.dozzle.icon=none`.
- Do not hand-edit upstream artwork beyond what the file needs to load. The WebP
  downscale in step 2 is the one exception.
- When the batch comes from GitHub issues, one PR per issue is fine; end each PR body
  with `Closes #<issue>`.
