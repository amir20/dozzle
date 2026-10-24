#!/usr/bin/env bash
# Renders a variant end to end: frames, soundtrack, poster baked into frame 0, mp4.
#   ./build.sh default            -> work/default/tour.mp4 and tour.jpg
#   ./build.sh default --publish  -> also writes the web encode to docs/public/video/
set -euo pipefail
cd "$(dirname "$0")"
V=${1:?usage: build.sh <variant> [--publish]}
W=work/$V
[[ -d work/shots ]] || { echo "no work/shots: run ./demo.sh, wait, then node capture.mjs" >&2; exit 1; }

node render.mjs "$V"
uv run -q --with numpy --with scipy python -I "variants/$V/music.py" "$PWD/$W/music.wav"

POSTER=$(node -e "
  const s = require('fs').readFileSync('variants/$V/index.html', 'utf8');
  console.log(String(Math.round(Number(s.match(/poster:\s*([\d.]+)/)[1]) * 30)).padStart(4, '0'));")
cp "$W/frames/f_$POSTER.jpg" "$W/tour.jpg"
cp "$W/tour.jpg" "$W/frames/f_0000.jpg" # replaces frame 0, so duration and sync stay the same

ffmpeg -loglevel error -y -framerate 30 -i "$W/frames/f_%04d.jpg" -i "$W/music.wav" \
  -af "loudnorm=I=-14:TP=-1.5:LRA=11" -c:v libx264 -preset slow -crf 17 -pix_fmt yuv420p \
  -movflags +faststart -c:a aac -b:a 192k -shortest "$W/tour.mp4"
echo "$W/tour.mp4"

if [[ "${2:-}" == "--publish" ]]; then
  P=../../docs/public/video
  ffmpeg -loglevel error -y -i "$W/tour.mp4" -vf scale=1600:-2 -c:v libx264 -preset slow -crf 28 -pix_fmt yuv420p \
    -movflags +faststart -c:a aac -b:a 96k "$P/dozzle-tour.mp4"
  ffmpeg -loglevel error -y -i "$W/tour.jpg" -vf scale=1280:-1 -q:v 4 "$P/dozzle-tour.jpg"
  echo "published to docs/public/video"
fi
