import type { RouteLocationRaw } from "vue-router";

/**
 * What slice of a container's log a view shows. A range with no end stays live:
 * "the last 15 minutes" keeps following, so it is the live route with a start,
 * and only a range that ends at a fixed time freezes into the historical view.
 */
export type TimeRange =
  | { kind: "live" }
  | { kind: "since"; since: Date; relative?: RelativeSpan }
  | { kind: "range"; from: Date; until: Date };

export const RELATIVE_SPANS = {
  "15m": 15 * 60_000,
  "1h": 60 * 60_000,
  "6h": 6 * 60 * 60_000,
  "24h": 24 * 60 * 60_000,
} as const;

export type RelativeSpan = keyof typeof RELATIVE_SPANS;

const isRelative = (value: string): value is RelativeSpan => value in RELATIVE_SPANS;

const validDate = (value: unknown) => {
  if (typeof value !== "string" || value === "") return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date;
};

/**
 * `?since=` on the live route: a relative span (`15m`), resolved against now
 * when the view opens so a bookmark keeps meaning "the last 15 minutes", or an
 * absolute time.
 */
export function parseSince(value: unknown, now = Date.now()): TimeRange {
  if (typeof value === "string" && isRelative(value)) {
    return { kind: "since", since: new Date(now - RELATIVE_SPANS[value]), relative: value };
  }
  const since = validDate(value);
  return since ? { kind: "since", since } : { kind: "live" };
}

/** `/container/:id/time/:datetime?until=`; without `until` it is a single moment, not a range. */
export function parseRange(datetime: string, until: unknown): TimeRange | undefined {
  const from = validDate(datetime);
  const end = validDate(until);
  return from && end && end > from ? { kind: "range", from, until: end } : undefined;
}

export function timeRangeRoute(containerId: string, range: TimeRange): RouteLocationRaw {
  switch (range.kind) {
    case "live":
      return { name: "/container/[id]", params: { id: containerId } };
    case "since":
      return {
        name: "/container/[id]",
        params: { id: containerId },
        query: { since: range.relative ?? range.since.toISOString() },
      };
    case "range":
      return {
        name: "/container/[id].time.[datetime]",
        params: { id: containerId, datetime: range.from.toISOString() },
        query: { until: range.until.toISOString() },
      };
  }
}

/** `from`/`to` for a request that should hold what the view holds (download, copy). */
export function appendRangeParams(params: URLSearchParams, range: TimeRange) {
  if (range.kind === "range") {
    params.append("from", range.from.toISOString());
    params.append("to", range.until.toISOString());
  } else if (range.kind === "since") {
    params.append("from", range.since.toISOString());
  }
}

/**
 * A range of `span` centered on a moment, for jumping to lines an empty view
 * points at. Never ends in the future.
 */
export function rangeAround(at: Date, span: number, now = Date.now()): TimeRange {
  const until = Math.min(now, at.getTime() + span / 2);
  return { kind: "range", from: new Date(until - span), until: new Date(until) };
}

/**
 * A time of day as a person types it: `9`, `9:05`, `09:05:30`, `9:05 pm`.
 * Returns hours, minutes and seconds on a 24-hour clock.
 */
export function parseTime(text: string): { h: number; m: number; s: number } | undefined {
  const match = text.trim().match(/^(\d{1,2})(?::(\d{2}))?(?::(\d{2}))?\s*([ap]\.?m\.?)?$/i);
  if (!match) return undefined;
  let h = Number(match[1]);
  const m = Number(match[2] ?? 0);
  const s = Number(match[3] ?? 0);
  const meridiem = match[4]?.[0].toLowerCase();
  if (meridiem) {
    if (h < 1 || h > 12) return undefined;
    h = (h % 12) + (meridiem === "p" ? 12 : 0);
  }
  if (h > 23 || m > 59 || s > 59) return undefined;
  return { h, m, s };
}

const pad = (n: number) => String(n).padStart(2, "0");

/** `15:04:05`, the form the time fields show and parseTime reads back. */
export function formatTime(d: Date): string {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

const sameDay = (a: Date, b: Date) => a.toDateString() === b.toDateString();

/** A range compactly: times alone when it is today, the day too otherwise. */
export function formatRange(from: Date, until: Date, now = new Date()): string {
  const time = (d: Date) => d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  const day = (d: Date) => d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  if (sameDay(from, until)) {
    return sameDay(from, now) ? `${time(from)} – ${time(until)}` : `${day(from)}, ${time(from)} – ${time(until)}`;
  }
  return `${day(from)} ${time(from)} – ${day(until)} ${time(until)}`;
}

/** A span the way the chip prints it: `45s`, `26m`, `1h 30m`, `2d 4h`. */
export function formatSpan(ms: number): string {
  const s = Math.max(0, Math.round(ms / 1000));
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) return m % 60 ? `${h}h ${m % 60}m` : `${h}h`;
  const d = Math.floor(h / 24);
  return h % 24 ? `${d}d ${h % 24}h` : `${d}d`;
}
