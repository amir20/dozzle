// Drives the sandboxed Dozzle from demo.sh and saves the real UI, state by state, to
// work/shots. Each sequence is a screenshot per keystroke or per stream flush.
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";
import path from "node:path";
const D = path.join(path.dirname(new URL(import.meta.url).pathname), "work", "shots");
mkdirSync(D, { recursive: true });
const URL_ = `http://localhost:${process.env.PORT ?? 3190}/`;
const b = await chromium.launch();
const ctx = await b.newContext({ viewport: { width: 1440, height: 810 }, deviceScaleFactor: 2, colorScheme: "dark" });
const p = await ctx.newPage();
const shot = (n) => p.screenshot({ path: `${D}/${n}.png` });
const pad = (i) => String(i).padStart(2, "0");
await p.goto(URL_);
await p.waitForTimeout(3000);
const fl = p.getByText("Finish later");
if (await fl.isVisible()) {
  await fl.click();
  await p.waitForTimeout(800);
}
await p.mouse.move(1300, 780);
await p.waitForTimeout(1500);
await shot("home");
// search
await p.keyboard.press("Meta+k");
await p.waitForTimeout(500);
await shot("search_00");
const q = "orders";
for (let i = 0; i < q.length; i++) {
  await p.keyboard.type(q[i]);
  await p.waitForTimeout(180);
  await shot("search_" + pad(i + 1));
}
await p.waitForTimeout(400);
await shot("search_" + pad(q.length + 1));
await p.keyboard.press("Enter");
await p.waitForTimeout(2500);
await p.mouse.move(1300, 780);
for (let i = 0; i < 24; i++) {
  await shot("worker_" + pad(i));
  await p.waitForTimeout(220);
}
console.log("worker", p.url());
// SQL
await p.goto(URL_);
await p.waitForTimeout(2000);
await p.getByText("checkout-api").first().click();
await p.waitForTimeout(3500);
await p.mouse.move(1300, 780);
await p.waitForTimeout(500);
await shot("api");
await p.keyboard.press("s");
await p.waitForTimeout(7000);
await p.mouse.move(1300, 780);
await shot("sql_00");
const ed = p.locator(".cm-content").first();
await ed.click();
await p.keyboard.press("ControlOrMeta+a");
await p.keyboard.press("Backspace");
const sql =
  "SELECT path, count(*) AS hits, round(avg(duration_ms)) AS avg_ms FROM logs WHERE path IS NOT NULL GROUP BY path ORDER BY avg_ms DESC";
let k = 1;
for (let i = 0; i < sql.length; i += 3) {
  await p.keyboard.insertText(sql.slice(i, i + 3));
  await p.waitForTimeout(60);
  await shot("sql_" + pad(k++));
}
await p.getByText("Query", { exact: true }).first().click();
await p.waitForTimeout(300);
await ed.click();
await p.keyboard.press("End");
await p.waitForTimeout(200);
await p.keyboard.press("ControlOrMeta+Enter");
await p.mouse.move(1300, 780);
await p.waitForTimeout(2500);
await shot("sql_" + pad(k++));
console.log("sql frames", k);
await b.close();
