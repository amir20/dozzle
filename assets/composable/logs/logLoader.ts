import { ShallowRef, type Ref } from "vue";
import { type LogMessage, LogEntry, LoadMoreLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import { loadBetween } from "./loadBetween";
import { notOnScreen } from "./logWindow";
import { useAlertMerger, isStreamLog } from "@/composable/cloud/alertMerger";

// Matches the rolling window size used for stats history
const LOG_WINDOW_FOR_DELTA = 300;

// What one windowed fetch returns at most when it is not asked for a minimum
// (defaultFetchSize in internal/web/logs_fetch.go). Fewer means the window held
// no more than that.
export const FETCH_PAGE = 500;

/**
 * Where loading older lines continues from: the latest, across the containers that
 * have not reached their start, of each one's oldest line on screen. Below it every
 * container's lines are on screen. A container with no line on screen says nothing
 * about where that is, so it does not count.
 */
export function olderCursor(
  oldestByContainer: Map<string, LogEntry<LogMessage>>,
  reachedStart: Set<string>,
): LogEntry<LogMessage> | undefined {
  let cursor: LogEntry<LogMessage> | undefined;
  for (const [id, oldest] of oldestByContainer) {
    if (reachedStart.has(id)) continue;
    if (!cursor || oldest.date > cursor.date) cursor = oldest;
  }
  return cursor;
}

/**
 * Merges older lines, sorted by time, into the window by time. A line older than
 * the cursor can still belong between lines already on screen, from a container
 * whose own oldest line is older still, so the lines cannot just be prepended.
 */
export function mergeByTime(messages: LogEntry<LogMessage>[], lines: LogEntry<LogMessage>[]): LogEntry<LogMessage>[] {
  if (lines.length === 0) return messages;
  const merged: LogEntry<LogMessage>[] = [];
  let i = 0;
  for (const m of messages) {
    const at = m.date.getTime();
    while (i < lines.length && lines[i].date.getTime() <= at) merged.push(lines[i++]);
    merged.push(m);
  }
  while (i < lines.length) merged.push(lines[i++]);
  return merged;
}

/** The containers that sent fewer lines than asked for, so the window widened back to their start. */
export function reachedStartIn(containers: Container[], logs: LogEntry<LogMessage>[], asked: number): Set<string> {
  const counts = new Map<string, number>();
  for (const l of logs) counts.set(l.containerID, (counts.get(l.containerID) ?? 0) + 1);
  return new Set(containers.filter((c) => (counts.get(c.id) ?? 0) < asked).map((c) => c.id));
}

export function useLogLoader(
  messages: ShallowRef<LogEntry<LogMessage>[]>,
  containers: Ref<Container[]>,
  params: Ref<URLSearchParams>,
  loadingMore: Ref<boolean>,
  {
    floor,
    startEdge,
    loadedOlder,
  }: {
    // The start the person picked; nothing older is loaded.
    floor?: Ref<Date | undefined>;
    // The row that replaces the loader once the floor is reached.
    startEdge?: () => LogEntry<LogMessage>;
    // Called after each load of older lines, with the containers whose every line
    // back to their start (or the floor) is now on screen.
    loadedOlder?: (reachedStart: Set<string>) => void;
  } = {},
) {
  const { withAlerts, decorateVisible } = useAlertMerger(messages, containers, params);
  // The containers whose start is on screen, for each window. A fresh window
  // (another view, filter or stream) opens under a new load-more row.
  const reachedStartByLoader = new WeakMap<LoadMoreLogEntry, Set<string>>();

  async function loadOlderLogs(entry: LoadMoreLogEntry) {
    if (!(messages.value[0] instanceof LoadMoreLogEntry)) throw new Error("No loadMoreLogEntry on first item");
    if (containers.value.length === 0) return;

    const [loader, ...existingLogs] = messages.value;
    if (existingLogs.length === 0) return;

    let reachedStart = reachedStartByLoader.get(entry);
    if (!reachedStart) reachedStartByLoader.set(entry, (reachedStart = new Set()));
    // Containers already at their start would only send back what is on screen.
    const pending = containers.value.filter((c) => !reachedStart.has(c.id));
    if (pending.length === 0) return;

    const containerIDs = new Set(containers.value.map((c) => c.id));
    const oldestByContainer = new Map<string, LogEntry<LogMessage>>();
    const countByContainer = new Map<string, number>();
    const nthByContainer = new Map<string, LogEntry<LogMessage>>();
    for (const log of existingLogs) {
      const id = log.containerID;
      // Rows that are not lines, like an update marker dated when its container
      // started, say nothing about how far back the lines on screen go.
      if (!id || !containerIDs.has(id) || !isStreamLog(log)) continue;
      if (!oldestByContainer.has(id)) {
        oldestByContainer.set(id, log);
      }
      const count = (countByContainer.get(id) ?? 0) + 1;
      countByContainer.set(id, count);
      if (count <= LOG_WINDOW_FOR_DELTA) {
        nthByContainer.set(id, log);
      }
    }

    const cursor = olderCursor(oldestByContainer, reachedStart);
    const to = cursor?.date ?? (existingLogs.find(isStreamLog) ?? existingLogs[0]).date;

    // With a floor, every load asks for [floor, cursor] at once. Docker scans its
    // log from the start for any window, so a narrower one costs the same and could
    // come back empty above lines that exist, while a `min` would widen the window
    // past the floor.
    const start = floor?.value;
    const min = start ? undefined : Math.ceil(100 / containers.value.length);
    let from = start;
    if (!from) {
      const nth = cursor && nthByContainer.get(cursor.containerID);
      const delta = to.getTime() - (nth?.date ?? to).getTime();
      from = new Date(to.getTime() + (delta !== 0 ? delta : -60_000));
    }

    try {
      loadingMore.value = true;
      // lastSeenId pins the cursor's own container to the line on screen, and
      // widens `to` a little, which only brings back lines already on screen.
      const { logs, signal } = await loadBetween(pending, params, from, to, { min, lastSeenId: cursor?.id });
      if (signal.aborted) return;

      for (const id of reachedStartIn(pending, logs, min ?? FETCH_PAGE)) reachedStart.add(id);
      const older = logs.filter(notOnScreen(existingLogs));
      const reachedFloor = start !== undefined && containers.value.every((c) => reachedStart.has(c.id));
      const head = reachedFloor && startEdge ? startEdge() : loader;
      if (older.length > 0 || head !== loader) {
        messages.value = [head, ...mergeByTime(existingLogs, await withAlerts(older))];
      }
      loadedOlder?.(new Set(reachedStart));
    } catch (err) {
      console.error(err);
    } finally {
      loadingMore.value = false;
    }
  }

  async function loadSkippedLogs(entry: SkippedLogsEntry) {
    if (containers.value.length === 0) return;

    const from = entry.firstSkipped.date;
    const to = entry.lastSkippedLog.date;

    try {
      loadingMore.value = true;
      const { logs, signal } = await loadBetween(containers.value, params, from, to, {
        lastSeenId: entry.lastSkippedLog.id,
      });
      if (signal.aborted) return;
      // lastSeenId widens `to` for every container, past the skipped run.
      const skipped = logs.filter((l) => l.date <= to).filter(notOnScreen(messages.value));

      if (skipped.length > 0) {
        const withAlertsApplied = await withAlerts(skipped);
        const updated = messages.value.flatMap((log) => (log === entry ? withAlertsApplied : [log]));
        messages.value = updated.length > config.maxLogs ? updated.slice(-config.maxLogs) : updated;
      }
    } catch (err) {
      console.error(err);
    } finally {
      loadingMore.value = false;
    }
  }

  return { loadOlderLogs, loadSkippedLogs, decorateWithAlerts: decorateVisible };
}
