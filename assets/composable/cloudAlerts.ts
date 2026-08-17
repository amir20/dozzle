import { useCloudConfig } from "@/composable/cloudConfig";
import { AlertLogEntry, CloudEventLogEntry, LogEntry, type LogMessage } from "@/models/LogEntry";

/**
 * One alert anchored inside a scroll window, as returned by
 * /api/cloud/alerts. Mirrors cloud.AlertHit on the Go side.
 */
export interface CloudAlert {
  alertId: number;
  containerId: string;
  hostId: string;
  /**
   * Dozzle's FNV-32a hash of the line that triggered the alert — the same id
   * LogEntry.id carries. Absent for metric and event alerts, which have no
   * triggering line and are positioned by `ts` alone.
   */
  logId?: number;
  /** Anchor time, unix nanoseconds. */
  ts: number;
  headline: string;
  level: string;
  eventCount: number;
  suppressedCount?: number;
  /** > 1 means the incident is wider than the container being viewed. */
  containerCount?: number;
  /** The alert's description, shown in Details. Present even when triage
   *  wrote no investigation. */
  summary?: string;
  investigation?: string;
  triageAction?: string;
  createdAt: number;
  lastActivityAt?: number;
  /**
   * True where the incident FIRST fired. Cloud appends every folded batch's
   * events to the original alert, so one incident can have activity in many
   * windows; follow-up anchors render as a compact marker rather than a card.
   */
  isOrigin: boolean;
  url?: string;
}

/** One matched event, as returned by /api/cloud/alerts?events=1. */
export interface CloudEvent {
  ts: number;
  logId?: number;
  containerId: string;
  level?: string;
  message?: string;
  type?: string;
  /** Human-readable summary; the only renderable text a non-log event has. */
  detail?: string;
  alertId?: number;
  suppressed: boolean;
}

interface CloudAlertsResponse {
  hits: CloudAlert[];
  events?: CloudEvent[];
  truncated?: boolean;
}

/**
 * fetchAlerts asks Cloud which alerts fired on these containers inside a
 * window. Gated on `linked` only — NOT on streamLogs, which log search
 * requires: alerts live in Cloud's database rather than the log store, so they
 * exist for anyone who linked cloud and configured a subscription.
 *
 * Never throws. This rides the scroll path, where the log lines have already
 * loaded and alerts are a decoration on top — a cloud outage must degrade to
 * "no alerts", never to "no logs".
 */
export async function fetchAlerts(
  containerIDs: string[],
  from: Date,
  to: Date,
  {
    linked,
    signal,
    followUps,
    events,
  }: { linked: boolean; signal?: AbortSignal; followUps?: boolean; events?: boolean },
): Promise<{ alerts: CloudAlert[]; events: CloudEvent[] }> {
  const empty = { alerts: [], events: [] };
  if (!linked || containerIDs.length === 0) return empty;

  const params = new URLSearchParams({
    containerIds: containerIDs.join(","),
    from: String(from.getTime() * 1_000_000),
    to: String(to.getTime() * 1_000_000),
  });
  if (followUps) params.set("followUps", "1");
  if (events) params.set("events", "1");

  try {
    const res = await fetch(withBase(`/api/cloud/alerts?${params}`), { signal });
    if (!res.ok) return empty;
    const body = (await res.json()) as CloudAlertsResponse;
    return { alerts: body.hits ?? [], events: body.events ?? [] };
  } catch {
    return empty;
  }
}

/**
 * useCloudAlerts exposes the fetch bound to the app's cloud-linked state, so
 * callers don't each have to read the shared config.
 */
export function useCloudAlerts() {
  const { cloudConfig } = useCloudConfig();
  const available = computed(() => !!cloudConfig.value?.linked);

  return {
    available,
    fetchAlerts: (
      containerIDs: string[],
      from: Date,
      to: Date,
      opts: { followUps?: boolean; events?: boolean; signal?: AbortSignal } = {},
    ) => fetchAlerts(containerIDs, from, to, { linked: available.value, ...opts }),
  };
}

/**
 * mergeAlerts splices alerts into a time-sorted run of log entries.
 *
 * An alert whose trigger line is present lands immediately AFTER that line,
 * matched on id — not by timestamp. Sorting alone is not enough: an alert
 * shares a millisecond with the line that caused it, so a stable sort would
 * place it either side by luck of input order, and the whole point is to show
 * which line tripped the rule.
 *
 * Alerts with no matching line (metric alerts, or a trigger line outside the
 * loaded range) fall back to timestamp position.
 *
 * `seen` carries anchor keys already placed by earlier loads, so overlapping
 * scroll windows don't duplicate. It is mutated as alerts are placed.
 */
export function mergeAlerts(
  logs: LogEntry<LogMessage>[],
  alerts: CloudAlert[],
  seen: Set<string>,
): LogEntry<LogMessage>[] {
  if (alerts.length === 0) return logs;

  // Origins only. A follow-up anchor marks an incident that was already open,
  // and the per-line badges say that with more precision — drawing a second
  // block for it would claim a delivery that never happened.
  const fresh = alerts.filter((a) => a.isOrigin && !seen.has(anchorKey(a)));
  if (fresh.length === 0) return logs;

  const byLogId = new Map<number, CloudAlert[]>();
  const byTime: CloudAlert[] = [];
  for (const alert of fresh) {
    // Only trust logId when that line is actually in this run; otherwise the
    // alert would silently vanish rather than fall back to its timestamp.
    if (alert.logId && logs.some((l) => l.id === alert.logId && l.containerID === alert.containerId)) {
      const bucket = byLogId.get(alert.logId);
      if (bucket) bucket.push(alert);
      else byLogId.set(alert.logId, [alert]);
    } else {
      byTime.push(alert);
    }
  }

  const place = (alert: CloudAlert) => {
    seen.add(anchorKey(alert));
    return new AlertLogEntry(alert, new Date(alert.ts / 1_000_000));
  };

  byTime.sort((a, b) => a.ts - b.ts);
  const merged: LogEntry<LogMessage>[] = [];
  let pending = 0;

  for (const log of logs) {
    // Timestamp-anchored alerts go in ahead of the first line at or past them.
    while (pending < byTime.length && byTime[pending].ts / 1_000_000 <= log.date.getTime()) {
      merged.push(place(byTime[pending++]));
    }
    merged.push(log);

    const triggered = byLogId.get(log.id);
    if (triggered) {
      for (const alert of triggered) {
        if (alert.containerId === log.containerID) merged.push(place(alert));
      }
      byLogId.delete(log.id);
    }
  }

  while (pending < byTime.length) merged.push(place(byTime[pending++]));

  return merged;
}

function anchorKey(alert: CloudAlert): string {
  return `${alert.alertId}:${alert.ts}`;
}

/**
 * Marks the log lines Cloud reports an event on.
 *
 * Matching is by log id — the FNV-32a hash both sides stamp on the same raw
 * line — so a line only gets badged when it really is the one that matched.
 * Metric and event notifications carry no line and are skipped rather than
 * guessed at by timestamp.
 *
 * Returns whether anything changed, so the caller can skip re-rendering the
 * list when it did not.
 */
export function attachEvents(logs: LogEntry<LogMessage>[], events: CloudEvent[]): boolean {
  if (events.length === 0) return false;

  // Only suppressed log events are badged. One that produced an alert is
  // already represented by the alert block, and a second marker on the same
  // line would say the same thing twice.
  const byLogId = new Map<number, CloudEvent>();
  for (const e of events) {
    if (e.logId && e.suppressed && isLogEvent(e)) byLogId.set(e.logId, e);
  }
  if (byLogId.size === 0) return false;

  let changed = false;
  for (const log of logs) {
    if (log.matchedEvent) continue;
    const event = byLogId.get(log.id);
    if (!event || event.containerId !== log.containerID) continue;
    log.matchedEvent = { alertId: event.alertId ?? 0, suppressed: event.suppressed, level: event.level ?? "" };
    changed = true;
  }
  return changed;
}

/** A log event has a line in the stream; metric and container events do not. */
export function isLogEvent(event: CloudEvent): boolean {
  return (event.type ?? "log") === "log";
}

/**
 * Splices metric and container events into the stream as their own rows.
 *
 * These have no log line to badge — they are the notification itself — so
 * unlike log events they cannot be shown by marking something already on
 * screen. Positioned by timestamp, since there is no id to anchor to.
 *
 * Events that produced an alert are skipped: the alert block already stands
 * for them.
 */
export function mergeCloudEvents(
  logs: LogEntry<LogMessage>[],
  events: CloudEvent[],
  seen: Set<string>,
): LogEntry<LogMessage>[] {
  const fresh = events.filter((e) => !isLogEvent(e) && e.suppressed && !seen.has(eventKey(e)));
  if (fresh.length === 0) return logs;

  fresh.sort((a, b) => a.ts - b.ts);
  const merged: LogEntry<LogMessage>[] = [];
  let pending = 0;

  const place = (event: CloudEvent) => {
    seen.add(eventKey(event));
    return new CloudEventLogEntry(event, new Date(event.ts / 1_000_000));
  };

  for (const log of logs) {
    while (pending < fresh.length && fresh[pending].ts / 1_000_000 <= log.date.getTime()) {
      merged.push(place(fresh[pending++]));
    }
    merged.push(log);
  }
  while (pending < fresh.length) merged.push(place(fresh[pending++]));

  return merged;
}

function eventKey(event: CloudEvent): string {
  return `event:${event.containerId}:${event.ts}`;
}
