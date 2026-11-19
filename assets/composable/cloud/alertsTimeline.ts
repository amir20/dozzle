import type { AlertBucket, CloudAlert, CloudEvent } from "./cloudAlerts";
import { dayBucket } from "@/utils/date";

/**
 * The alerts panel as one timeline: what the reader's rules caught, a day at a
 * time. A row is a problem, and how often it happened and how often the reader
 * was told about it are two numbers on that row. The matches that told nobody
 * anything are one quiet row per day, so they are there to look at without
 * reading as twelve more problems.
 *
 * Cloud speaks in alerts (one notification each) and events (each match a rule
 * made). Nothing here leaks that split to the reader.
 */

/** The windows the panel offers, and how finely each is charted before the panel has a width. */
export const TIMELINE_WINDOWS = {
  "24h": { hours: 24, buckets: 24 },
  "7d": { hours: 168, buckets: 28 },
} as const;
export type TimelineWindow = keyof typeof TIMELINE_WINDOWS;
/** Cloud's cap on buckets per call (maxAlertBuckets in internal/web/cloud_alerts.go). */
export const MAX_BUCKETS = 200;

/** One match, as listed under an expanded row. */
export interface Occurrence {
  ts: number;
  containerId: string;
  logId?: number;
  text: string;
}

interface RowBase {
  /** Unique across the panel: the day plus what the row is about. */
  key: string;
  containers: string[];
  happened: number;
  /** Newest occurrence, unix nanoseconds. */
  latest: number;
  /** Newest first. */
  occurrences: Occurrence[];
}

export interface ProblemRow extends RowBase {
  kind: "problem";
  headline: string;
  level: string;
  /** Notifications that went out for this problem on this day. 0 means it was
   *  told earlier, and these are more of the same. */
  notified: number;
  /** The newest alert's page in Dozzle Cloud, when Cloud sent one. */
  url?: string;
}

export interface QuietRow extends RowBase {
  kind: "quiet";
}

export type TimelineRow = ProblemRow | QuietRow;

export interface TimelineDay {
  key: number;
  label: string;
  rows: TimelineRow[];
}

const SEVERITY: Record<string, number> = { fatal: 4, error: 3, warn: 2, warning: 2, info: 1 };
const worse = (a: string, b: string) => ((SEVERITY[b] ?? 0) > (SEVERITY[a] ?? 0) ? b : a);

/** Two alerts of the same rule with the same headline are the same problem
 *  firing again, so they share a row. */
const problemKey = (hit: CloudAlert) => `${hit.subscriptionId ?? ""}|${hit.headline}`;

const isRuleMatch = (e: CloudEvent) => e.type !== "deploy";

type Draft = {
  row: TimelineRow;
  containers: Set<string>;
  origins: Set<string>;
  withEvents: Set<string>;
  urlTs?: number;
};

/**
 * Splits a window's alerts and matches into days, then rows.
 *
 * `hits` are Cloud's alert anchors (followUps included, so an alert still
 * matching today carries its headline even if it first fired yesterday), and
 * `events` the matches. Days come newest first, rows inside a day by their
 * newest occurrence, and the day's quiet row last.
 */
export function buildTimeline(
  hits: CloudAlert[],
  events: CloudEvent[],
  locale: string | undefined,
  now: Date = new Date(),
): TimelineDay[] {
  const byAlert = new Map<string, CloudAlert>();
  for (const hit of hits) {
    const known = byAlert.get(hit.alertId);
    // Any anchor carries the headline; the newest is the one Cloud wrote last.
    if (!known || hit.ts > known.ts) byAlert.set(hit.alertId, hit);
  }

  const days = new Map<number, { key: number; label: string; drafts: Map<string, Draft> }>();
  function draftFor(ts: number, key: string, make: () => TimelineRow): Draft | undefined {
    const bucket = dayBucket(new Date(ts / 1e6), locale, now);
    if (!bucket) return undefined;
    let day = days.get(bucket.key);
    if (!day) days.set(bucket.key, (day = { ...bucket, drafts: new Map() }));
    let draft = day.drafts.get(key);
    if (!draft) {
      const row = make();
      row.key = `${bucket.key}:${key}`;
      draft = { row, containers: new Set(), origins: new Set(), withEvents: new Set() };
      day.drafts.set(key, draft);
    }
    return draft;
  }

  const problem = (hit: { headline: string; level: string }) => (): ProblemRow => ({
    kind: "problem",
    key: "",
    headline: hit.headline,
    level: hit.level || "info",
    containers: [],
    happened: 0,
    notified: 0,
    latest: 0,
    occurrences: [],
  });
  const quiet = (): QuietRow => ({ kind: "quiet", key: "", containers: [], happened: 0, latest: 0, occurrences: [] });

  function add(draft: Draft, occurrence: Occurrence) {
    draft.row.occurrences.push(occurrence);
    draft.containers.add(occurrence.containerId);
    draft.row.latest = Math.max(draft.row.latest, occurrence.ts);
  }

  for (const event of events) {
    if (!isRuleMatch(event)) continue;
    const occurrence: Occurrence = {
      ts: event.ts,
      containerId: event.containerId,
      logId: event.logId,
      text: event.message || event.detail || "",
    };
    const hit = event.alertId ? byAlert.get(event.alertId) : undefined;
    if (!event.alertId) {
      const draft = draftFor(event.ts, "quiet", quiet);
      if (draft) add(draft, occurrence);
      continue;
    }
    // A match on an alert whose anchor is not in the window still told the
    // reader about it once; the line itself stands in for the headline.
    const key = hit ? problemKey(hit) : `alert:${event.alertId}`;
    const draft = draftFor(event.ts, key, problem(hit ?? { headline: occurrence.text, level: event.level ?? "info" }));
    if (!draft) continue;
    add(draft, occurrence);
    draft.withEvents.add(event.alertId);
  }

  for (const hit of hits) {
    const draft = draftFor(hit.ts, problemKey(hit), problem(hit));
    if (!draft) continue;
    const row = draft.row as ProblemRow;
    row.level = worse(row.level, hit.level || "info");
    if (hit.isOrigin) draft.origins.add(hit.alertId);
    if (hit.url && hit.ts >= (draft.urlTs ?? 0)) {
      row.url = hit.url;
      draft.urlTs = hit.ts;
    }
    draft.containers.add(hit.containerId);
    // An alert with no matches of its own on this day (an older Cloud, or a
    // metric alert) is still something that happened: the anchor is the moment.
    if (!draft.withEvents.has(hit.alertId)) {
      add(draft, { ts: hit.ts, containerId: hit.containerId, logId: hit.logId, text: hit.headline });
      // Counted from the alert itself only when there are no matches to count.
      if (!events.length) row.happened += Math.max(1, hit.eventCount) - 1;
    }
  }

  return [...days.values()]
    .sort((a, b) => b.key - a.key)
    .map((day) => {
      const rows = [...day.drafts.values()].map(({ row, containers, origins }) => {
        row.containers = [...containers];
        row.happened += row.occurrences.length;
        row.occurrences.sort((a, b) => b.ts - a.ts);
        if (row.kind === "problem") row.notified = origins.size;
        return row;
      });
      const problems = rows.filter((r) => r.kind === "problem").sort((a, b) => b.latest - a.latest);
      return { key: day.key, label: day.label, rows: [...problems, ...rows.filter((r) => r.kind === "quiet")] };
    });
}

/** Where each bucket ends, in ns: the next one's start, the last at the window's end. */
export function bucketEnd(buckets: AlertBucket[], index: number, to: number) {
  return buckets[index + 1]?.startTs ?? to;
}

/**
 * The row a stretch of time belongs to: the one with the most matches inside
 * it, problems before the quiet row on a tie. With nothing inside it (a match
 * the list never got, past Cloud's cap) the row closest in time stands in, so a
 * click always lands somewhere.
 */
export function rowAt(days: TimelineDay[], from: number, to: number): TimelineRow | undefined {
  let best: TimelineRow | undefined;
  let bestCount = 0;
  let nearest: TimelineRow | undefined;
  let nearestGap = Infinity;
  for (const day of days) {
    for (const row of day.rows) {
      const count = row.occurrences.filter((o) => o.ts >= from && o.ts < to).length;
      if (count > bestCount || (count === bestCount && count > 0 && best?.kind === "quiet" && row.kind === "problem")) {
        best = row;
        bestCount = count;
      }
      for (const o of row.occurrences) {
        const gap = o.ts < from ? from - o.ts : o.ts >= to ? o.ts - to : 0;
        if (gap < nearestGap) {
          nearestGap = gap;
          nearest = row;
        }
      }
    }
  }
  return best ?? nearest;
}

/** The problem a stretch of time is mostly about, for the hover readout. */
export function problemAt(days: TimelineDay[], from: number, to: number): string | undefined {
  const row = rowAt(days, from, to);
  if (row?.kind !== "problem") return undefined;
  return row.occurrences.some((o) => o.ts >= from && o.ts < to) ? row.headline : undefined;
}

/** The severity colour of the worst notification sent inside a stretch of time. */
export function notifiedLevel(hits: CloudAlert[], from: number, to: number) {
  return hits
    .filter((h) => h.isOrigin && h.ts >= from && h.ts < to)
    .reduce((level, h) => worse(level, h.level || "info"), "info");
}

/** The tinted circle behind an alert's glyph. Severity rides the icon. */
export function levelTint(level?: string) {
  switch (level) {
    case "error":
    case "fatal":
      return "bg-error/10 text-error";
    case "warn":
    case "warning":
      return "bg-warning/10 text-warning";
    default:
      return "bg-info/10 text-info";
  }
}

/** The status pill an alert's level wears. */
export function levelPill(level?: string) {
  switch (level) {
    case "error":
    case "fatal":
      return "status-pill-error";
    case "warn":
    case "warning":
      return "status-pill-warning";
    default:
      return "status-pill-neutral";
  }
}

/** A solid dot in the level's colour. */
export function levelDot(level?: string) {
  switch (level) {
    case "error":
    case "fatal":
      return "bg-error";
    case "warn":
    case "warning":
      return "bg-warning";
    default:
      return "bg-info";
  }
}
