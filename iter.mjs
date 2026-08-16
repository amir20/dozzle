import { chromium } from "@playwright/test";
const b = await chromium.launch();
const p = await b.newPage({ viewport: { width: Number(process.env.W || 1700), height: 900 }, colorScheme: "dark" });
await p.goto("http://localhost:3100/container/5735440523fe?alertPreview=1", { waitUntil: "domcontentloaded" });
await p.waitForSelector(".alert-row", { timeout: 15000 });
const el = await p.$(".alert-row");
await el.scrollIntoViewIfNeeded();
await p.waitForTimeout(500);
if (process.env.EXPAND) { await p.click(".alert-row button"); await p.waitForTimeout(300); }
const bb = await el.boundingBox();
await p.screenshot({ path: process.env.SHOT, clip: { x: Math.max(0, bb.x - 240), y: Math.max(0, bb.y - 70), width: Math.min(1420, (Number(process.env.W || 1700)) - Math.max(0, bb.x - 240)), height: process.env.EXPAND ? 240 : 150 } });
console.log("captured, row height:", Math.round(bb.height));
await b.close();
