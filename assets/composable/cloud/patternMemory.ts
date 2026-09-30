import type { LogEntry, LogMessage, PatternMemory } from "@/models/LogEntry";

/**
 * Error memory: whether an error or warn line's pattern is new for its
 * container. Dozzle names each line by (container id, log id) — the FNV-32a
 * hash both sides stamp on the same raw line — and Cloud answers from what it
 * remembers, so no pattern is ever re-derived here.
 */

/** Levels a line needs to be worth asking about. */
const MEMORY_LEVELS = new Set(["error", "fatal", "critical", "severe", "warn", "warning"]);

/** Cloud's per-request cap. The newest lines win when a window holds more. */
export const MAX_PATTERN_LINES = 500;

/**
 * How many times an unanswered line is asked about. A line Cloud has no answer
 * for is usually one that hasn't reached its log store yet, so it gets a few
 * polls to arrive — then stops costing a request every poll forever.
 */
const MAX_ASKS = 3;
const asks = new WeakMap<LogEntry<LogMessage>, number>();

/**
 * The statuses this release shows. Cloud returns them all; the chip starts
 * with NEW alone, the one claim precise enough to ship first.
 */
export const SHOWN_STATUSES: ReadonlySet<PatternMemory["status"]> = new Set(["new"]);

/** One pattern as /api/cloud/patterns returns it. Mirrors cloud.PatternContext. */
export interface PatternContextHit {
  containerId: string;
  logIds: number[];
  pattern: string;
  level: string;
  status: PatternMemory["status"];
  firstSeen?: number;
  daysSeen?: number;
  ratePerHour: number;
  usualRatePerHour: number;
}

/** The lines in a window still worth asking Cloud about, newest first. */
export function linesNeedingMemory(logs: LogEntry<LogMessage>[]): LogEntry<LogMessage>[] {
  const out: LogEntry<LogMessage>[] = [];
  for (let i = logs.length - 1; i >= 0 && out.length < MAX_PATTERN_LINES; i--) {
    const log = logs[i];
    if (!log.id || !log.level || !MEMORY_LEVELS.has(log.level)) continue;
    if (log.patternMemory || (asks.get(log) ?? 0) >= MAX_ASKS) continue;
    out.push(log);
  }
  return out;
}

/**
 * Asks Cloud about these lines. Never throws: memory is a decoration on lines
 * that have already rendered, so any failure degrades to "no chips".
 */
export async function fetchPatternContext(
  lines: LogEntry<LogMessage>[],
  from: Date,
  to: Date,
  signal?: AbortSignal,
): Promise<PatternContextHit[]> {
  if (lines.length === 0) return [];
  for (const l of lines) asks.set(l, (asks.get(l) ?? 0) + 1);
  try {
    const res = await fetch(withBase("/api/cloud/patterns"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        lines: lines.map((l) => ({ containerId: l.containerID, logId: l.id })),
        from: from.getTime() * 1_000_000,
        to: to.getTime() * 1_000_000,
      }),
      signal,
    });
    if (res.status !== 200) return [];
    const body = (await res.json()) as { patterns?: PatternContextHit[] };
    return body.patterns ?? [];
  } catch {
    return [];
  }
}

/**
 * Marks each line with its pattern's memory, matched on container and log id.
 * Returns whether any line changed, so the caller can skip a re-render.
 */
export function attachPatternMemory(logs: LogEntry<LogMessage>[], hits: PatternContextHit[]): boolean {
  if (hits.length === 0) return false;
  const byLine = new Map<string, PatternContextHit>();
  for (const h of hits) {
    for (const id of h.logIds) byLine.set(`${h.containerId}:${id}`, h);
  }
  let changed = false;
  for (const log of logs) {
    const hit = byLine.get(`${log.containerID}:${log.id}`);
    if (!hit || log.patternMemory) continue;
    log.patternMemory = {
      status: hit.status,
      pattern: hit.pattern,
      firstSeen: hit.firstSeen,
      daysSeen: hit.daysSeen,
      ratePerHour: hit.ratePerHour,
      usualRatePerHour: hit.usualRatePerHour,
    };
    if (SHOWN_STATUSES.has(hit.status)) trackUsage("memory.chip.shown");
    changed = true;
  }
  return changed;
}

/** The placeholders VictoriaLogs' `collapse_nums prettify` writes into a pattern. */
const PLACEHOLDER = /<(?:N|IP4|UUID|TIME|DATE|DATETIME|W)>/g;

/**
 * A regex matching the lines of a pattern, for "show only these lines".
 *
 * This is the one place a pattern is turned back into a matcher. It only
 * drives a search filter, so if it ever drifts from how Cloud collapses lines
 * the worst case is a filter that misses some lines — never a mislabelled one.
 */
export function patternToRegex(pattern: string): string {
  return pattern
    .split(PLACEHOLDER)
    .map((part) => part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
    .join(".+?");
}
