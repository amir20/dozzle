import { ShallowRef, type Ref } from "vue";
import { type LogMessage, LogEntry, LoadMoreLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import { loadBetween } from "./loadBetween";
import { useAlertMerger, isStreamLog } from "@/composable/cloud/alertMerger";

// Matches the rolling window size used for stats history
const LOG_WINDOW_FOR_DELTA = 300;

// What one windowed fetch returns at most when it is not asked for a minimum
// (defaultFetchSize in internal/web/logs_fetch.go). Fewer means the window held
// no more than that.
export const FETCH_PAGE = 500;

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

  async function loadOlderLogs(entry: LoadMoreLogEntry) {
    if (!(messages.value[0] instanceof LoadMoreLogEntry)) throw new Error("No loadMoreLogEntry on first item");
    if (containers.value.length === 0) return;

    const [loader, ...existingLogs] = messages.value;
    if (existingLogs.length === 0) return;

    const containerIDs = new Set(containers.value.map((c) => c.id));
    const earliestByContainer = new Map<string, LogEntry<LogMessage>>();
    const countByContainer = new Map<string, number>();
    const nthByContainer = new Map<string, LogEntry<LogMessage>>();
    for (const log of existingLogs) {
      const id = log.containerID;
      // Rows that are not lines, like an update marker dated when its container
      // started, say nothing about how far back the lines on screen go.
      if (!id || !containerIDs.has(id) || !isStreamLog(log)) continue;
      if (!earliestByContainer.has(id)) {
        earliestByContainer.set(id, log);
      }
      const count = (countByContainer.get(id) ?? 0) + 1;
      countByContainer.set(id, count);
      if (count <= LOG_WINDOW_FOR_DELTA) {
        nthByContainer.set(id, log);
      }
    }

    // With a floor, every load asks for [floor, earliest on screen] at once. Docker
    // scans its log from the start for any window, so a narrower one costs the same
    // and could come back empty above lines that exist, while a `min` would widen
    // the window past the floor.
    const start = floor?.value;
    if (start) {
      try {
        loadingMore.value = true;
        const to = existingLogs.find(isStreamLog)?.date ?? existingLogs[0].date;
        const results = await Promise.all(
          containers.value.map((c) =>
            loadBetween(c, params, start, to, { lastSeenId: earliestByContainer.get(c.id)?.id }),
          ),
        );
        if (results.some(({ signal }) => signal.aborted)) return;
        const older = results.flatMap(({ logs }) => logs).sort((a, b) => a.date.getTime() - b.date.getTime());
        const reachedFloor = results.every(({ logs }) => logs.length < FETCH_PAGE);
        const head = reachedFloor && startEdge ? startEdge() : loader;
        if (older.length > 0 || head !== loader) {
          messages.value = [head, ...(await withAlerts(older)), ...existingLogs];
        }
        loadedOlder?.(new Set(containers.value.filter((_, i) => results[i].logs.length < FETCH_PAGE).map((c) => c.id)));
      } catch (err) {
        console.error(err);
      } finally {
        loadingMore.value = false;
      }
      return;
    }

    try {
      loadingMore.value = true;
      const minPerContainer = Math.ceil(100 / containers.value.length);
      const firstOnScreen = (existingLogs.find(isStreamLog) ?? existingLogs[0]).date;

      const results = await Promise.all(
        containers.value.map((c) => {
          const earliest = earliestByContainer.get(c.id);
          const to = earliest?.date ?? firstOnScreen;
          const nth = nthByContainer.get(c.id);
          const delta = to.getTime() - (nth?.date ?? to).getTime();
          const from = new Date(to.getTime() + (delta !== 0 ? delta : -60_000));
          return loadBetween(c, params, from, to, {
            min: minPerContainer,
            lastSeenId: earliest?.id,
          });
        }),
      );

      const allNewLogs = results
        .filter(({ signal }) => !signal.aborted)
        .flatMap(({ logs }) => logs)
        .sort((a, b) => a.date.getTime() - b.date.getTime());

      if (allNewLogs.length > 0) {
        messages.value = [loader, ...(await withAlerts(allNewLogs)), ...existingLogs];
      }
      // Fewer than asked for means the window widened back to the container's start.
      loadedOlder?.(
        new Set(
          containers.value
            .filter((_, i) => !results[i].signal.aborted && results[i].logs.length < minPerContainer)
            .map((c) => c.id),
        ),
      );
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
    const ownerContainerID = entry.lastSkippedLog.containerID;

    try {
      loadingMore.value = true;
      const results = await Promise.all(
        containers.value.map((c) => {
          const lastSeenId = c.id === ownerContainerID ? entry.lastSkippedLog.id : undefined;
          return loadBetween(c, params, from, to, { lastSeenId });
        }),
      );
      const allLogs = results
        .filter(({ signal }) => !signal.aborted)
        .flatMap(({ logs }) => logs)
        .sort((a, b) => a.date.getTime() - b.date.getTime());

      if (allLogs.length > 0) {
        const withAlertsApplied = await withAlerts(allLogs);
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
