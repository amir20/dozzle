// Drives the sandboxed Dozzle from demo.sh and saves the real UI, state by state, to
// work/shots. Each sequence is a screenshot per keystroke or per stream flush, and
// work/shots/manifest.js records how many frames each one got, for the compositions.
//
// The chat sequence needs the sandbox linked to Dozzle Cloud; it is skipped otherwise.
import { chromium } from "playwright";
import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";

const D = path.join(path.dirname(new URL(import.meta.url).pathname), "work", "shots");
mkdirSync(D, { recursive: true });
const URL_ = `http://localhost:${process.env.PORT ?? 3190}/`;
const QUESTION = process.env.CHAT_QUESTION ?? "Why does email.send keep failing?";

const b = await chromium.launch();
const ctx = await b.newContext({ viewport: { width: 1440, height: 810 }, deviceScaleFactor: 2, colorScheme: "dark" });
const p = await ctx.newPage();
const manifest = {};
const pad = (i) => String(i).padStart(2, "0");
const shot = (n) => p.screenshot({ path: `${D}/${n}.png` });
const park = () => p.mouse.move(1430, 800);
async function seq(name, n, ms) {
  for (let i = 0; i < n; i++) {
    await shot(`${name}_${pad(i)}`);
    await p.waitForTimeout(ms);
  }
  manifest[name] = n;
}
async function typeSeq(name, text, step, ms = 60) {
  let k = 1;
  for (let i = 0; i < text.length; i += step) {
    await p.keyboard.insertText(text.slice(i, i + step));
    await p.waitForTimeout(ms);
    await shot(`${name}_${pad(k++)}`);
  }
  manifest[name] = k;
}
async function dismiss() {
  for (const t of ["Finish later", "Got it"]) {
    const l = p.getByText(t, { exact: true }).first();
    if (await l.isVisible().catch(() => false)) {
      await l.click();
      await p.waitForTimeout(500);
    }
  }
}

// dashboard, then the updates waiting in the megaphone
await p.goto(URL_);
await p.waitForTimeout(4000);
await dismiss();
await park();
await p.waitForTimeout(1500);
await shot("home");
await p.locator("svg.icon-wiggle").first().click();
await p.waitForTimeout(900);
await shot("mega");
await p
  .getByText(/available/i)
  .first()
  .click();
await p.waitForTimeout(2500);
await park();
await shot("updates");
await p.keyboard.press("Escape");
await p.waitForTimeout(800);

// fuzzy search into a live stream
await p.goto(URL_);
await p.waitForTimeout(2500);
await park();
await p.keyboard.press("ControlOrMeta+k");
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
manifest.search = q.length + 2;
await p.keyboard.press("Enter");
await p.waitForTimeout(2500);
await park();
await seq("worker", 24, 220);

// every container in one stream
await p.goto(URL_ + "group/shop");
await p.waitForTimeout(3500);
await park();
await seq("group", 20, 250);

// SQL over checkout-api's JSON logs
await p.goto(URL_);
await p.waitForTimeout(2000);
await p.getByText("checkout-api").first().click();
await p.waitForTimeout(3500);
await park();
await shot("api");
await p.keyboard.press("s");
await p.waitForTimeout(7000);
await park();
await shot("sql_00");
const ed = p.locator(".cm-content").first();
await ed.click();
await p.keyboard.press("ControlOrMeta+a");
await p.keyboard.press("Backspace");
await typeSeq(
  "sql",
  "SELECT path, count(*) AS hits, round(avg(duration_ms)) AS avg_ms FROM logs WHERE path IS NOT NULL GROUP BY path ORDER BY avg_ms DESC",
  3,
);
await p.getByText("Query", { exact: true }).first().click();
await p.waitForTimeout(300);
await ed.click();
await p.keyboard.press("End");
await p.waitForTimeout(200);
await p.keyboard.press("ControlOrMeta+Enter");
await park();
await p.waitForTimeout(2500);
await shot("sql_result");

// one click from a log line to a chart of that field
await p.keyboard.press("Escape");
await p.waitForTimeout(1200);
await p.getByText("payment provider timeout").last().click();
await p.waitForTimeout(1200);
await park();
await shot("details");
const chartBtn = p.locator('.chart-btn[title*="duration_ms"]').first();
await chartBtn.hover();
await p.waitForTimeout(400);
await shot("details_hover");
await chartBtn.click();
await p.waitForTimeout(7000);
await park();
await shot("chart");

// Ask Dozzle, if this sandbox is linked to Cloud
await p.keyboard.press("Escape");
await p.goto(URL_);
await p.waitForTimeout(2000);
await p.getByText("orders-worker").first().click();
await p.waitForTimeout(3000);
const ask = p.locator('[title="Ask Dozzle"]').first();
if (await ask.isVisible().catch(() => false)) {
  await ask.click();
  await p.waitForTimeout(1500);
  await park();
  await shot("chat_00");
  await p.getByPlaceholder(/Ask about/).click();
  await typeSeq("chat", QUESTION, 2, 50);
  await p.keyboard.press("Enter");
  await park();
  await seq("chat_reply", 60, 500);
}

writeFileSync(path.join(D, "manifest.js"), `window.SHOTS = ${JSON.stringify(manifest)};\n`);
console.log(manifest);
await b.close();
