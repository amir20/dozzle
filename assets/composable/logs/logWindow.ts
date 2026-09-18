import { type JSONObject, type LogMessage, LogEntry, SkippedLogsEntry } from "@/models/LogEntry";

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
