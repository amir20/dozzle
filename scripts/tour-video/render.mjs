// Renders a variant's index.html to JPEG frames. Every frame is window.setTime(t),
// so a frame is a pure function of time and a partial re-render matches the rest.
//   node render.mjs <variant> [from] [to]       frames, by index
//   node render.mjs <variant> --stills 1.5 8    single stills at those seconds
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";
import path from "node:path";

const dir = path.dirname(new URL(import.meta.url).pathname);
const [variant, ...rest] = process.argv.slice(2);
const out = path.join(dir, "work", variant, rest[0] === "--stills" ? "stills" : "frames");
mkdirSync(out, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1920, height: 1080 } });
await page.goto("file://" + path.join(dir, "variants", variant, "index.html"));
await page.evaluate(() => window.ready);
const { duration } = await page.evaluate(() => window.meta);

const frames =
  rest[0] === "--stills"
    ? rest.slice(1).map((s) => ({ t: Number(s), name: `still_${s}.jpg` }))
    : Array.from({ length: duration * 30 }, (_, i) => ({
        t: i / 30,
        name: `f_${String(i).padStart(4, "0")}.jpg`,
      })).slice(Number(rest[0] ?? 0), Number(rest[1] ?? Infinity));

for (const { t, name } of frames) {
  await page.evaluate((t) => window.setTime(t), t);
  await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
  await page.screenshot({ path: path.join(out, name), type: "jpeg", quality: 93 });
}
await browser.close();
