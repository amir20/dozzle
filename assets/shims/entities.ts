// ansi-to-html pulls in `entities`, whose full HTML entity table is ~135 KB of the main
// bundle. We call it with escapeXML: false (see assets/utils/ansi.ts), so only encodeXML is
// ever reachable and the table is dead weight. Alias `entities` to this in vite.config.ts.
const XML_ESCAPES: Record<string, string> = {
  '"': "&quot;",
  "&": "&amp;",
  "'": "&apos;",
  "<": "&lt;",
  ">": "&gt;",
};

export const encodeXML = (str: string): string => str.replace(/["&'<>]/g, (c) => XML_ESCAPES[c]);

export default { encodeXML };
