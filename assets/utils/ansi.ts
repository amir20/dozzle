import { AnsiUp } from "ansi_up";

// The backend has already HTML-escaped the text and spliced in <a> and <mark>, so escaping
// is off. Classes (`ansi-red-fg`, `ansi-bright-blue-bg`, ...) map the 16 standard colors onto
// the theme-aware CSS variables in main.css; 256-color indices 16-255 and truecolor stay inline.
//
// AnsiUp carries SGR state and any truncated escape over to the next call, but every log line
// is converted on its own, so each call gets a fresh converter. Lines with no ESC at all, the
// vast majority, never build one.
const converter = (): AnsiUp => {
  const ansiUp = new AnsiUp();
  ansiUp.escape_html = false;
  ansiUp.use_classes = true;
  // Dim has never rendered here, so keep it invisible rather than fading the line.
  ansiUp.faintStyle = "";
  return ansiUp;
};

export const colorize = (value: string): string => (value.includes("\x1b") ? converter().ansi_to_html(value) : value);
