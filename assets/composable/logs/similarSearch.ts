import stripAnsi from "strip-ansi";
import { ComplexLogEntry, GroupedLogEntry, LogEntry, SimpleLogEntry, type JSONObject } from "@/models/LogEntry";

/**
 * The parts of a log line that change from one occurrence to the next. Order
 * matters: the alternation takes the first branch that matches at a position,
 * so a timestamp or UUID is claimed whole before its digits look like numbers.
 */
const VARIABLE = new RegExp(
  [
    /\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?/, // ISO-ish timestamp
    /\d{2}:\d{2}:\d{2}(?:[.,]\d+)?/, // bare clock time
    /\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b/, // UUID
    /\b\d{1,3}(?:\.\d{1,3}){3}(?::\d+)?\b/, // IPv4, optional port
    /\b(?=[0-9a-f]*\d)[0-9a-f]{7,}\b/, // hex id or hash
    /\b\d+(?:\.\d+)?(?:ms|s|m|h|µs|ns|kb|mb|gb|b|%)?\b/, // number, optional unit
  ]
    .map((r) => r.source)
    .join("|"),
  "gi",
);

// Past this the tail of a line is usually payload, and a pattern that long is
// as likely to miss on one stray value as it is to find anything.
const MAX_LENGTH = 160;

/** Keys a structured logger puts its human message under, most common first. */
const MESSAGE_KEYS = ["msg", "message", "@message", "log"];

/** The key holding a structured line's human message, if it has one. */
export function messageKeyOf(fields: Record<string, unknown>): string | undefined {
  return MESSAGE_KEYS.find((key) => typeof fields[key] === "string" && (fields[key] as string).trim());
}

export type SimilarSearch = {
  /** For Dozzle's own search: a Go-compatible regex with the variable parts wildcarded. */
  regex: string;
  /** For Cloud's word search: the longest stretch of the line that never varies. */
  phrase: string;
};

/**
 * Turn one line into a search for the lines like it. `whole` anchors the regex
 * at both ends, which is right when `text` is one JSON field's value (search
 * matches each value on its own) and wrong for a raw line, which may carry a
 * prefix the search sees but the reader does not.
 */
export function similarSearch(text: string, whole = false): SimilarSearch | null {
  // A line already under a search carries the server's highlight markers.
  const clean = stripAnsi(text)
    .replace(/[\uE000\uE001]/g, "")
    .trim();
  if (!clean) return null;
  const truncated = clean.length > MAX_LENGTH;
  // Cut on a space so the cut never lands inside a value and leaves half of it as fixed text.
  const head = clean.slice(0, MAX_LENGTH);
  const line = truncated ? head.slice(0, head.lastIndexOf(" ") > 0 ? head.lastIndexOf(" ") : MAX_LENGTH) : clean;

  const fixed: string[] = [];
  let regex = "";
  let last = 0;
  for (const match of line.matchAll(VARIABLE)) {
    fixed.push(line.slice(last, match.index));
    regex +=
      escapeRegex(line.slice(last, match.index)) + (/^\d+(?:\.\d+)?$/.test(match[0]) ? "\\d+(?:\\.\\d+)?" : "\\S+");
    last = match.index + match[0].length;
  }
  fixed.push(line.slice(last));
  regex += escapeRegex(line.slice(last));

  const phrase = fixed
    .map((part) => part.trim())
    .reduce((longest, part) => (part.length > longest.length ? part : longest), "");
  // Nothing worded survives: the line was all values, and every line would match.
  if ((phrase.match(/\p{L}/gu) ?? []).length < 3) return null;

  if (whole) regex = `^${regex}${truncated ? "" : "$"}`;
  return { regex, phrase };
}

/** The text of an entry worth matching on, and whether it is a whole field value. */
export function similarSource(entry: LogEntry<string | JSONObject>): { text: string; whole: boolean } | null {
  if (entry instanceof ComplexLogEntry) {
    const key = messageKeyOf(entry.unfilteredMessage);
    // Without a message field, the serialized object varies by every field and finds nothing.
    return key ? { text: entry.unfilteredMessage[key] as string, whole: true } : null;
  }
  if (entry instanceof GroupedLogEntry) return { text: entry.message[0] ?? "", whole: false };
  if (entry instanceof SimpleLogEntry) return { text: entry.rawMessage || entry.message || "", whole: false };
  return null;
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
