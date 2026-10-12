/**
 * @vitest-environment jsdom
 */
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, test } from "vitest";
import { colorize } from "./ansi";

// These assert on what the reader sees (text, color, weight) rather than on exact markup,
// so the converter behind `colorize` can change without rewriting every expectation.

const ESC = "\x1b";
const css = readFileSync(resolve(import.meta.dirname, "../main.css"), "utf8");

// A class the converter emits must resolve to a rule in main.css, or the color is lost.
const classRule = (cls: string, prop: "color" | "background-color"): string => {
  const match = new RegExp(`\\.${cls}\\s*\\{\\s*${prop}:\\s*([^;}]+)`).exec(css);
  if (!match) throw new Error(`no ${prop} rule for .${cls} in main.css`);
  return match[1].trim();
};

// The theme's own text and page colors are what an uncolored span shows anyway.
const DEFAULTS = new Set(["var(--color-base-content)", "var(--color-base-100)"]);

type Look = { fg?: string; bg?: string; bold: boolean; italic: boolean; underline: boolean };

const render = (html: string): HTMLElement => {
  const root = document.createElement("div");
  root.innerHTML = html;
  return root;
};

/** Resolves the effective look of the first text node containing `needle`. */
const lookOf = (html: string, needle: string): Look => {
  const root = render(html);
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let node: Node | null = walker.nextNode();
  while (node && !node.textContent?.includes(needle)) node = walker.nextNode();
  if (!node) throw new Error(`"${needle}" not found in ${html}`);

  const look: Look = { bold: false, italic: false, underline: false };
  let weight: boolean | undefined;
  // Walk outward so the innermost declaration wins.
  for (let el = node.parentElement; el && el !== root; el = el.parentElement) {
    const tag = el.tagName.toLowerCase();
    if (weight === undefined) {
      if (el.style.fontWeight === "normal") weight = false;
      else if (tag === "b" || tag === "strong" || /bold|[6-9]00/.test(el.style.fontWeight)) weight = true;
    }
    look.bold = weight ?? false;
    if (tag === "i" || tag === "em" || el.style.fontStyle === "italic") look.italic = true;
    if (tag === "u" || el.style.textDecoration.includes("underline")) look.underline = true;

    let fg = el.style.color || undefined;
    let bg = el.style.backgroundColor || undefined;
    for (const cls of el.classList) {
      if (cls.endsWith("-fg")) fg ??= classRule(cls, "color");
      if (cls.endsWith("-bg")) bg ??= classRule(cls, "background-color");
    }
    if (look.fg === undefined && fg) look.fg = fg;
    if (look.bg === undefined && bg) look.bg = bg;
  }
  if (look.fg && DEFAULTS.has(look.fg)) look.fg = undefined;
  if (look.bg && DEFAULTS.has(look.bg)) look.bg = undefined;
  return { ...look, fg: look.fg?.replace(/\s+/g, ""), bg: look.bg?.replace(/\s+/g, "") };
};

const plain: Look = { bold: false, italic: false, underline: false };

describe("colorize: already-escaped backend output", () => {
  test("plain text is returned untouched", () => {
    expect(colorize("plain text")).toBe("plain text");
  });

  test("entities are not escaped a second time", () => {
    const out = colorize("&lt;b&gt;tag&lt;/b&gt; a &amp; b &quot;q&quot;");
    expect(out).not.toContain("&amp;lt;");
    expect(out).not.toContain("&amp;amp;");
    expect(out).not.toContain("&amp;quot;");
    expect(render(out).textContent).toBe('<b>tag</b> a & b "q"');
  });

  test("<mark> and <a href> markup survive as elements", () => {
    const html =
      '<mark>hit</mark> <a href="https://x.com/?a=1&amp;b=2" target="_blank" rel="noopener noreferrer external">https://x.com</a>';
    const root = render(colorize(html));
    expect(root.querySelector("mark")?.textContent).toBe("hit");
    const a = root.querySelector("a");
    expect(a?.getAttribute("href")).toBe("https://x.com/?a=1&b=2");
    expect(a?.getAttribute("target")).toBe("_blank");
    expect(a?.textContent).toBe("https://x.com");
  });

  test("markup inside a colored run keeps both the element and the color", () => {
    const out = colorize(`${ESC}[31m<mark>hit</mark> <a href="https://x.com">link</a>${ESC}[0m`);
    const root = render(out);
    expect(root.querySelector("mark")?.textContent).toBe("hit");
    expect(root.querySelector("a")?.getAttribute("href")).toBe("https://x.com");
    expect(lookOf(out, "hit").fg).toBe("var(--ansi-red)");
    expect(lookOf(out, "link").fg).toBe("var(--ansi-red)");
  });
});

describe("colorize: 16-color palette uses theme variables", () => {
  const names = ["black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"];

  test.each(names.map((n, i) => [n, i]))("fg %s", (name, i) => {
    expect(lookOf(colorize(`${ESC}[${30 + i}mX${ESC}[0m`), "X")).toEqual({ ...plain, fg: `var(--ansi-${name})` });
  });

  test.each(names.map((n, i) => [n, i]))("bg %s", (name, i) => {
    expect(lookOf(colorize(`${ESC}[${40 + i}mX${ESC}[0m`), "X")).toEqual({ ...plain, bg: `var(--ansi-${name})` });
  });

  test.each(names.map((n, i) => [n, i]))("bright fg %s", (name, i) => {
    expect(lookOf(colorize(`${ESC}[${90 + i}mX${ESC}[0m`), "X")).toEqual({
      ...plain,
      fg: `var(--ansi-bright-${name})`,
    });
  });

  test.each(names.map((n, i) => [n, i]))("bright bg %s", (name, i) => {
    expect(lookOf(colorize(`${ESC}[${100 + i}mX${ESC}[0m`), "X")).toEqual({
      ...plain,
      bg: `var(--ansi-bright-${name})`,
    });
  });
});

describe("colorize: extended colors", () => {
  test("256-color indices 0-15 map onto the theme palette", () => {
    expect(lookOf(colorize(`${ESC}[38;5;3mX${ESC}[0m`), "X").fg).toBe("var(--ansi-yellow)");
    expect(lookOf(colorize(`${ESC}[38;5;9mX${ESC}[0m`), "X").fg).toBe("var(--ansi-bright-red)");
  });

  test("256-color cube and grayscale keep xterm values", () => {
    expect(lookOf(colorize(`${ESC}[38;5;208mX${ESC}[0m`), "X").fg).toBe("rgb(255,135,0)");
    expect(lookOf(colorize(`${ESC}[48;5;21mX${ESC}[0m`), "X").bg).toBe("rgb(0,0,255)");
    expect(lookOf(colorize(`${ESC}[38;5;244mX${ESC}[0m`), "X").fg).toBe("rgb(128,128,128)");
  });

  test("truecolor fg and bg", () => {
    expect(lookOf(colorize(`${ESC}[38;2;100;150;250mX${ESC}[0m`), "X").fg).toBe("rgb(100,150,250)");
    expect(lookOf(colorize(`${ESC}[48;2;1;2;3mX${ESC}[0m`), "X").bg).toBe("rgb(1,2,3)");
  });
});

describe("colorize: text attributes and resets", () => {
  test("bold, italic, underline", () => {
    expect(lookOf(colorize(`${ESC}[1mX${ESC}[0m`), "X")).toEqual({ ...plain, bold: true });
    expect(lookOf(colorize(`${ESC}[3mX${ESC}[0m`), "X")).toEqual({ ...plain, italic: true });
    expect(lookOf(colorize(`${ESC}[4mX${ESC}[0m`), "X")).toEqual({ ...plain, underline: true });
  });

  test("bold combined with a color", () => {
    expect(lookOf(colorize(`${ESC}[1;32mX${ESC}[0m`), "X")).toEqual({ ...plain, bold: true, fg: "var(--ansi-green)" });
  });

  test("dim changes nothing visible", () => {
    const out = colorize(`${ESC}[2mX${ESC}[0m`);
    expect(lookOf(out, "X")).toEqual(plain);
    expect(render(out).querySelector("[style*=opacity]")).toBeNull();
  });

  test("reset mid-line ends the color", () => {
    const out = colorize(`${ESC}[31mred${ESC}[0m plain`);
    expect(lookOf(out, "red").fg).toBe("var(--ansi-red)");
    expect(lookOf(out, "plain")).toEqual(plain);
    expect(render(out).textContent).toBe("red plain");
  });

  test("39 resets only the foreground", () => {
    const out = colorize(`${ESC}[31;1mA${ESC}[39mB${ESC}[0m`);
    expect(lookOf(out, "A")).toEqual({ ...plain, bold: true, fg: "var(--ansi-red)" });
    expect(lookOf(out, "B")).toEqual({ ...plain, bold: true });
  });

  test("22 turns bold off", () => {
    const out = colorize(`${ESC}[1mA${ESC}[22mB`);
    expect(lookOf(out, "A").bold).toBe(true);
    expect(lookOf(out, "B").bold).toBe(false);
  });

  test("non-SGR control sequences are dropped", () => {
    expect(render(colorize(`${ESC}[K${ESC}[2Jclear`)).textContent).toBe("clear");
  });
});

describe("colorize: one line never leaks into the next", () => {
  test("an unterminated color does not carry over", () => {
    expect(lookOf(colorize(`${ESC}[31mopen`), "open").fg).toBe("var(--ansi-red)");
    expect(colorize("next")).toBe("next");
  });

  test("a truncated escape is dropped and does not eat the next line", () => {
    expect(render(colorize(`x${ESC}[`)).textContent).toBe("x");
    expect(colorize("next")).toBe("next");
  });
});
