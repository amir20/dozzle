#!/usr/bin/env bash
# Regenerates every raster icon and the docs copy of the logo from assets/logo.svg.
#
# The PNGs and the .ico are committed, so this is a maintainer tool rather than a
# build step. Run it after touching assets/logo.svg and commit whatever changes,
# otherwise the mark in the sidebar and the mark in the browser tab drift apart,
# which is how docs/public/logo.svg ended up a whole palette behind
# assets/logo.svg.
#
# Needs rsvg-convert (librsvg) and magick (ImageMagick 7).
set -euo pipefail

cd "$(dirname "$0")/.."

for tool in rsvg-convert magick; do
  command -v "$tool" >/dev/null || { echo "missing $tool" >&2; exit 1; }
done

# Apple masks the icon itself, so this ships full bleed with no baked-in corners.
readonly APPLE_BG="#343434"

render() { rsvg-convert -w "$2" -h "$2" "$1" -o "$3"; }

# logo.svg carries a safe margin for the rounded masks Apple and Android apply.
# Nothing masks a browser tab, so the tab icons get the margin trimmed back off
# and the mark fills the square instead of sitting at 86% of it.
render_full() {
  rsvg-convert -w 1024 -h 1024 "$1" -o "$tmp/full.png"
  magick "$tmp/full.png" -trim +repage -resize "$2x$2" \
    -background none -gravity center -extent "$2x$2" "$3"
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

render_full assets/logo.svg 192 public/favicon.png

render assets/logo.svg 512 "$tmp/apple.png"
magick "$tmp/apple.png" -background "$APPLE_BG" -alpha remove -alpha off public/apple-touch-icon.png

for size in 16 24 32 48; do
  render_full assets/logo.svg "$size" "$tmp/ico-$size.png"
done
magick "$tmp/ico-16.png" "$tmp/ico-24.png" "$tmp/ico-32.png" "$tmp/ico-48.png" public/favicon.ico

cp assets/logo.svg docs/public/logo.svg

echo "icons regenerated"
