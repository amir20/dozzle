#!/usr/bin/env node
// Translations have no Crowdin or Weblate behind them, so nothing would otherwise
// notice when an English page changes and its translations do not. Each translated
// file carries a sourceHash of the English page it was made from. This script fails
// when those disagree, which turns silent drift into a red build.
//
//   node docs/scripts/check-translations.mjs           verify
//   node docs/scripts/check-translations.mjs --update   re-stamp after translating

import { createHash } from "node:crypto";
import { readdir, readFile, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const DOCS = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const LOCALES = ["zh", "de", "fr", "es"];
const SKIP = new Set([".vitepress", "node_modules", "public", "scripts", "superpowers", ...LOCALES]);

const hash = (s) => createHash("sha256").update(s).digest("hex").slice(0, 12);

async function englishPages(dir = DOCS, acc = []) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (dir === DOCS && SKIP.has(entry.name)) continue;
      await englishPages(join(dir, entry.name), acc);
    } else if (entry.name.endsWith(".md")) {
      acc.push(relative(DOCS, join(dir, entry.name)));
    }
  }
  return acc;
}

function readStamp(text) {
  const m = text.match(/^---\n([\s\S]*?)\n---/);
  return m?.[1].match(/^sourceHash:\s*(\S+)\s*$/m)?.[1] ?? null;
}

function writeStamp(text, value) {
  const m = text.match(/^---\n([\s\S]*?)\n---/);
  if (!m) return `---\nsourceHash: ${value}\n---\n\n${text}`;
  const body = m[1].replace(/^sourceHash:.*\n?/m, "").trimEnd();
  return text.replace(m[0], `---\n${body}\nsourceHash: ${value}\n---`);
}

const update = process.argv.includes("--update");
const missing = [];
const stale = [];
let stamped = 0;

for (const page of await englishPages()) {
  const source = await readFile(join(DOCS, page), "utf8");
  const expected = hash(source);

  for (const locale of LOCALES) {
    const target = join(DOCS, locale, page);
    if (!existsSync(target)) {
      missing.push(`${locale}/${page}`);
      continue;
    }
    const text = await readFile(target, "utf8");
    if (readStamp(text) === expected) continue;
    if (update) {
      await writeFile(target, writeStamp(text, expected));
      stamped++;
    } else {
      stale.push(`${locale}/${page}`);
    }
  }
}

if (update) {
  console.log(`Stamped ${stamped} file(s).`);
  if (missing.length) {
    console.error(`\nStill missing ${missing.length} translation(s):`);
    for (const f of missing) console.error(`  docs/${f}`);
    process.exit(1);
  }
  process.exit(0);
}

if (!missing.length && !stale.length) {
  console.log("All translations are in sync with their English sources.");
  process.exit(0);
}

if (missing.length) {
  console.error(`Missing ${missing.length} translation(s):`);
  for (const f of missing) console.error(`  docs/${f}`);
}
if (stale.length) {
  console.error(`\n${stale.length} translation(s) are out of date with the English source:`);
  for (const f of stale) console.error(`  docs/${f}`);
}
console.error(
  `\nUpdate the translated page(s), then run:\n  node docs/scripts/check-translations.mjs --update\nto re-stamp them.`,
);
process.exit(1);
