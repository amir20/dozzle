# Tour video

The 33 second video behind "Watch the tour" on the dozzle.dev home page
(`docs/.vitepress/theme/components/TourVideo.vue`, files in `docs/public/video/`).

Everything after the opening terminal is the real Dozzle UI, captured from a sandboxed
instance that only sees five fictional containers. The UI fills the frame and a camera
zooms in on whatever is happening. Each frame is drawn by a browser page as a pure
function of time, then encoded with a synthesized soundtrack.

## Layout

- `demo.sh`: starts (or `down` removes) the fake containers and a filtered Dozzle on `:3190`,
  with actions on so the updates drawer shows. `gen/*.sh` are the log generators.
- `capture.mjs`: drives that Dozzle with Playwright and saves each UI state to `work/shots/`,
  plus `work/shots/manifest.js` with the frame count of each sequence.
- `variants/<name>/index.html`: the composition. `window.setTime(t)` draws frame `t`,
  `CAMERA` holds the per-scene framing, and `window.meta` the duration and poster time.
- `variants/<name>/music.py`: the soundtrack. Its cut times mirror the composition's.
- `render.mjs`, `build.sh`: frames, then audio, poster, and mp4.

`work/` is ignored. It holds captures, frames and renders, all reproducible, and
`work/data` holds the sandbox's Dozzle data, including its Cloud link.

## Rebuild

```bash
DOZZLE_IMAGE=amir20/dozzle:pr-1234 ./demo.sh   # a build with the UI you want to show
# link http://localhost:3190 to Dozzle Cloud: the chat scene needs it, and the default
# composition expects its frames, so an unlinked capture won't render
# wait ~3 minutes so the SQL query and chart have rows
node capture.mjs
./build.sh default --publish                   # also writes docs/public/video/
./demo.sh down
```

Needs Docker, ffmpeg, `uv` (the soundtrack runs with numpy and scipy), and the repo's
Playwright Chromium. The chat scene asks a real question, so it uses Cloud usage.

To check framing without a full render: `node render.mjs default --stills 8 18.5 26`
writes stills to `work/default/stills/`.

## Editing

- A fresh capture can move things: the error block scrolls with the stream, and panels
  shift when the UI changes. Recheck `CAMERA`, the SQL highlight box, and `REPLY_END`
  (the first `chat_reply_NN` frame where the answer is complete) against stills.
- Keep stills from mid-transition in the review, and anything the viewer should read on
  screen for about 0.3 seconds a word.
