import AnsiConvertor from "ansi-to-html";

// Theme-aware ANSI palette. Values are CSS variables defined per DaisyUI theme in main.css,
// so log colors stay legible on both light and dark backgrounds. The `colors` map merges over
// the library's 256-color defaults, so indices 16-255 keep their standard values.
const ansiConvertor = new AnsiConvertor({
  escapeXML: false,
  fg: "var(--color-base-content)",
  bg: "var(--color-base-100)",
  colors: {
    0: "var(--ansi-black)",
    1: "var(--ansi-red)",
    2: "var(--ansi-green)",
    3: "var(--ansi-yellow)",
    4: "var(--ansi-blue)",
    5: "var(--ansi-magenta)",
    6: "var(--ansi-cyan)",
    7: "var(--ansi-white)",
    8: "var(--ansi-bright-black)",
    9: "var(--ansi-bright-red)",
    10: "var(--ansi-bright-green)",
    11: "var(--ansi-bright-yellow)",
    12: "var(--ansi-bright-blue)",
    13: "var(--ansi-bright-magenta)",
    14: "var(--ansi-bright-cyan)",
    15: "var(--ansi-bright-white)",
  },
});

export const colorize = (value: string): string => ansiConvertor.toHtml(value);
