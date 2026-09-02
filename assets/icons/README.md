# App icons

`apps/` holds a curated subset of [homarr-labs/dashboard-icons](https://github.com/homarr-labs/dashboard-icons),
vendored so Dozzle never has to fetch an icon at runtime. Requesting `sonarr.svg` from a CDN would
tell a third party what a user is running and would break air-gapped installs.

The dashboard-icons repository is MIT licensed. The logos themselves remain the property of their
respective projects and are used here to identify them.

## Conventions

- One file per icon, named after the upstream slug: `sonarr.svg`.
- SVG where upstream has one. WebP downscaled to 64px otherwise, plus a handful of SVGs that were
  large enough (jdownloader was 330KB) to be worth rasterizing.
- `<slug>-light` is artwork for dark backgrounds and `<slug>-dark` for light ones, matching the
  upstream convention. Both are optional; most icons read fine either way.

Resolution from image name to slug lives in `assets/utils/appIcons.ts`.

## Adding an icon

Drop the file in `apps/` and, if the image name does not match the slug, add an entry to `ALIASES`
in `appIcons.ts`. `vite.config.ts` keeps this directory out of the inline-asset budget so each icon
stays a separately fetched file.
