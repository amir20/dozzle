import { ShallowRef, type Ref } from "vue";
import { type LogMessage, LogEntry, LoadMoreLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import { loadBetween } from "@/composable/loadBetween";

// Matches the rolling window size used for stats history
const LOG_WINDOW_FOR_DELTA = 300;

export function useLogLoader(
  messages: ShallowRef<LogEntry<LogMessage>[]>,
  containers: Ref<Container[]>,
  params: Ref<URLSearchParams>,
  loadingMore: Ref<boolean>,
) {
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
      if (!id || !containerIDs.has(id)) continue;
      if (!earliestByContainer.has(id)) {
        earliestByContainer.set(id, log);
      }
      const count = (countByContainer.get(id) ?? 0) + 1;
      countByContainer.set(id, count);
      if (count <= LOG_WINDOW_FOR_DELTA) {
        nthByContainer.set(id, log);
      }
    }

    try {
      loadingMore.value = true;
      const minPerContainer = Math.ceil(100 / containers.value.length);

      const results = await Promise.all(
        containers.value.map((c) => {
          const earliest = earliestByContainer.get(c.id);
          const to = earliest?.date ?? existingLogs[0].date;
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
        messages.value = [loader, ...allNewLogs, ...existingLogs];
      }
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
        const updated = messages.value.flatMap((log) => (log === entry ? allLogs : [log]));
        messages.value = updated.length > config.maxLogs ? updated.slice(-config.maxLogs) : updated;
      }
    } catch (err) {
      console.error(err);
    } finally {
      loadingMore.value = false;
    }
  }

  return { loadOlderLogs, loadSkippedLogs };
}
