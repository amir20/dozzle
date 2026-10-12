// `entities` ships the full HTML entity table (~135 KB), and markdown-it pulls it in for
// decodeHTMLStrict alone, so `entities` is aliased to this in vite.config.ts.
//
// markdown-it decodes numeric references itself and calls decodeHTMLStrict only
// for named ones, treating "not a known entity" as "leave the text alone" — so a short table
// costs nothing but the long tail of named entities, which renders literally instead.

// The entities an assistant answer or a log line realistically contains.
const NAMED: Record<string, string> = {
  amp: "&",
  lt: "<",
  gt: ">",
  quot: '"',
  apos: "'",
  nbsp: " ",
  hellip: "…",
  mdash: "—",
  ndash: "–",
  lsquo: "‘",
  rsquo: "’",
  ldquo: "“",
  rdquo: "”",
  laquo: "«",
  raquo: "»",
  bull: "•",
  middot: "·",
  times: "×",
  divide: "÷",
  plusmn: "±",
  deg: "°",
  micro: "µ",
  para: "¶",
  sect: "§",
  copy: "©",
  reg: "®",
  trade: "™",
  euro: "€",
  pound: "£",
  yen: "¥",
  cent: "¢",
  dagger: "†",
  ne: "≠",
  le: "≤",
  ge: "≥",
  infin: "∞",
  larr: "←",
  rarr: "→",
  uarr: "↑",
  darr: "↓",
  harr: "↔",
};

/** Decodes one named or numeric reference, returning the input untouched when it
 *  is neither — the same contract the real `decodeHTMLStrict` has. */
export const decodeHTMLStrict = (str: string): string => {
  const match = /^&(#x[a-f0-9]{1,8}|#[0-9]{1,8}|[a-z][a-z0-9]{1,31});$/i.exec(str);
  if (!match) return str;

  const name = match[1];
  if (name[0] === "#") {
    const code = name[1].toLowerCase() === "x" ? parseInt(name.slice(2), 16) : parseInt(name.slice(1), 10);
    return Number.isNaN(code) ? str : String.fromCodePoint(code);
  }

  return NAMED[name] ?? str;
};

export default { decodeHTMLStrict };
