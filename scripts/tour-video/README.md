# Tour video

The 20 second video behind "Watch the tour" on the dozzle.dev home page
(`docs/.vitepress/theme/components/TourVideo.vue`, files in `docs/public/video/`).

Everything after the opening terminal is the real Dozzle UI, captured from a sandboxed
instance that only sees five fictional busybox containers. Each frame is drawn by a
browser page as a pure function of time, then encoded with a synthesized soundtrack.

## Layout

- `demo.sh`: starts (or `down` removes) the fake containers and a filtered Dozzle on `:3190`.
  `gen/*.sh` are the log generators they run.
- `capture.mjs`: drives that Dozzle with Playwright and saves each UI state to `work/shots/`.
- `variants/<name>/index.html`: the composition. `window.setTime(t)` draws frame `t`, and
  `window.meta` holds the duration and the poster time.
- `variants/<name>/music.py`: the soundtrack for that cut.
- `render.mjs`, `build.sh`: frames, then audio, poster, and mp4.

`work/` is ignored. It holds captures, frames and renders, all reproducible.

## Rebuild

```bash
./demo.sh                     # wait ~3 minutes so the SQL query has rows
node capture.mjs              # only needed when the UI changed
./build.sh default            # or: ./build.sh deadpan
./build.sh default --publish  # also writes docs/public/video/
./demo.sh down
```

Needs Docker, ffmpeg, `uv` (the soundtrack runs with numpy and scipy), and the repo's
Playwright Chromium.

To check a layout without a full render: `node render.mjs default --stills 1.5 8 15`
writes stills to `work/default/stills/`.

## Editing

- Copy or timing changes only touch `index.html`. Re-render, re-encode.
- The SQL highlight box and the zoom origins are in screenshot coordinates. Recheck them
  after a fresh capture, since the layout of the real UI may have moved.
- Keep stills from mid-transition in the review. A plain crossfade between two busy
  frames reads as mud, so both cuts dip through the background instead.
