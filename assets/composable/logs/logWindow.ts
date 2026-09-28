import {
  type JSONObject,
  type LogMessage,
  ComplexLogEntry,
  GroupedLogEntry,
  LogEntry,
  SimpleLogEntry,
  SkippedLogsEntry,
} from "@/models/LogEntry";

export type AppendBatchOptions = {
  maxLogs: number;
  // The user has scrolled up, so the view must not move under them.
  paused: boolean;
  loadSkipped: (entry: SkippedLogsEntry) => Promise<void>;
};

/**
 * Appends a flushed batch to the view, keeping it within maxLogs. While paused an
 * overflowing batch collapses into one SkippedLogsEntry the user can expand later;
 * otherwise the oldest lines drop off the top.
 *
 * Growing an existing SkippedLogsEntry mutates it in place and returns the same
 * array, since the entry's count is its own ref.
 */
export function appendBatch(
  messages: LogEntry<LogMessage>[],
  batch: LogEntry<LogMessage>[],
  { maxLogs, paused, loadSkipped }: AppendBatchOptions,
): LogEntry<LogMessage>[] {
  if (messages.length + batch.length > maxLogs) {
    if (paused) {
      const firstItem = batch.at(0) as LogEntry<string | JSONObject>;
      const lastItem = batch.at(-1) as LogEntry<string | JSONObject>;
      const last = messages.at(-1);
      if (last instanceof SkippedLogsEntry) {
        last.addSkippedEntries(batch.length, lastItem);
        return messages;
      }
      return [...messages, new SkippedLogsEntry(new Date(), batch.length, firstItem, lastItem, loadSkipped)];
    }
    if (batch.length > maxLogs / 2) {
      return batch.slice(-maxLogs / 2);
    }
    return [...messages, ...batch].slice(-maxLogs);
  }
  return [...messages, ...batch];
}

const lineKey = (e: LogEntry<LogMessage>) => `${e.containerID}:${e.id}:${e.date.getTime()}`;

/**
 * Every connect replays the last lines of each container, so a stream reopened under a
 * view that is still on screen sends back what the reader already has. Returns a test
 * that keeps only what comes after the last line seen from that entry's container.
 *
 * The cutoff is per container: in a merged view one container can lag another, and a
 * single cutoff would drop the lagging one's lines. Lines sharing the cutoff's
 * millisecond are told apart by id, which is a hash of the content.
 */
export function newerThanOnScreen(messages: LogEntry<LogMessage>[]): (entry: LogEntry<LogMessage>) => boolean {
  const last = new Map<string, { ts: number; keys: Set<string> }>();
  const see = (e: LogEntry<LogMessage>) => {
    const ts = e.date.getTime();
    const seen = last.get(e.containerID);
    if (!seen || ts > seen.ts) last.set(e.containerID, { ts, keys: new Set([lineKey(e)]) });
    else if (ts === seen.ts) seen.keys.add(lineKey(e));
  };
  for (const m of messages) {
    if (m instanceof SimpleLogEntry || m instanceof GroupedLogEntry || m instanceof ComplexLogEntry) see(m);
    // Collapsed while the reader was scrolled up, but already accounted for.
    else if (m instanceof SkippedLogsEntry) see(m.lastSkippedLog);
  }

  return (entry) => {
    const seen = last.get(entry.containerID);
    if (!seen) return true;
    const ts = entry.date.getTime();
    return ts > seen.ts || (ts === seen.ts && !seen.keys.has(lineKey(entry)));
  };
}

/** The time of the newest line the view holds, collapsed ones included. */
export function newestOnScreen(messages: LogEntry<LogMessage>[]): number {
  let newest = 0;
  for (const m of messages) {
    const line = m instanceof SkippedLogsEntry ? m.lastSkippedLog : m;
    if (line instanceof SimpleLogEntry || line instanceof GroupedLogEntry || line instanceof ComplexLogEntry) {
      newest = Math.max(newest, line.date.getTime());
    }
  }
  return newest;
}

/** Keeps backfilled lines that are not already on screen. */
export function notOnScreen(messages: LogEntry<LogMessage>[]): (entry: LogEntry<LogMessage>) => boolean {
  const keys = new Set(messages.map(lineKey));
  return (entry) => !keys.has(lineKey(entry));
}
