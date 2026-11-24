// Reads a structured line the way a person scans it: what it says, then if it
// was a request, how it went. Used by the "Readable" line style; "Raw" keeps the
// line's own field order and never calls this.
//
// The same keys Dozzle Cloud reads (ui/app/utils/log-reading.ts there), so a
// line reads the same in both.

const MESSAGE_KEYS = ["msg", "message", "event", "event.action"];
const METHOD_KEYS = ["method", "http.method", "http.request.method"];
const PATH_KEYS = ["path", "url.path", "http.target", "url"];
const STATUS_KEYS = ["status", "http.status", "http.response.status_code", "statusCode"];
const ERROR_KEYS = ["error", "err", "error.message", "exception", "exc_info", "reason"];
// Each unit's keys, and what one of it is in milliseconds.
const DURATION_KEYS: Array<[string[], number]> = [
  [["event.duration", "duration_ns", "latency_ns", "durationNanos"], 1e-6],
  [["duration_us", "latency_us", "duration_micros"], 1e-3],
  [["duration_ms", "latency_ms", "responseTime", "response_time_ms", "took_ms", "took", "elapsed_ms"], 1],
  [["duration_s", "latency_s", "request_time", "elapsed"], 1000],
  [["duration", "latency"], 1],
];

/** Fields that repeat what the row already shows: its timestamp and its level marker. */
export const TIME_KEYS = ["time", "ts", "timestamp", "@timestamp", "t"];
export const LEVEL_KEYS = ["level", "lvl", "severity", "log.level"];

export interface Reading {
  title?: string;
  method?: string;
  path?: string;
  status?: number;
  durationMs?: number;
  error?: string;
  /** The keys read into the parts above, so they aren't repeated among the fields. */
  used: Set<string>;
}

export function readFields(fields: Record<string, unknown>): Reading {
  const used = new Set<string>();
  const str = (keys: string[]) => {
    for (const k of keys) {
      const v = fields[k];
      if (typeof v === "string" && v !== "") {
        used.add(k);
        return v;
      }
    }
    return undefined;
  };
  const num = (keys: string[]) => {
    for (const k of keys) {
      const v = fields[k];
      const n = typeof v === "number" ? v : typeof v === "string" && /^\d+(\.\d+)?$/.test(v) ? Number(v) : NaN;
      if (Number.isFinite(n)) return { k, n };
    }
    return undefined;
  };

  const r: Reading = { used };
  r.title = str(MESSAGE_KEYS);
  r.error = str(ERROR_KEYS);

  // Method and status make it a request. Path and duration only join the title
  // then: on any other line they are ordinary fields and stay where they were.
  const method = str(METHOD_KEYS);
  const status = num(STATUS_KEYS);
  if (method !== undefined || status !== undefined) {
    r.method = method;
    if (status) {
      r.status = status.n;
      used.add(status.k);
    }
    r.path = str(PATH_KEYS);
    for (const [keys, factor] of DURATION_KEYS) {
      const d = num(keys);
      if (d) {
        r.durationMs = d.n * factor;
        used.add(d.k);
        break;
      }
    }
  }
  return r;
}

export function formatDuration(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms >= 10) return `${Math.round(ms)}ms`;
  return `${ms.toFixed(1)}ms`;
}

/**
 * What a line says, for folding repeats: lines in a row with the same key fold
 * into one. A structured line's message and error; anything else, its text.
 */
export function repeatKey(fields: Record<string, unknown>): string | undefined {
  const r = readFields(fields);
  if (r.title === undefined && r.error === undefined) return undefined;
  return `${r.title ?? ""}\u0000${r.error ?? ""}`;
}
