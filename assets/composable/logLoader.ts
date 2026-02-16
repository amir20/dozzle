import { ShallowRef, type Ref } from "vue";
import { type LogMessage, LogEntry, LoadMoreLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import { loadBetween } from "@/composable/loadBetween";

export function useLogLoader(
  messages: ShallowRef<LogEntry<LogMessage>[]>,
  container: Ref<Container> | undefined,
  containers: Ref<Container[]>,
  params: Ref<URLSearchParams>,
  loadingMore: Ref<boolean>,
) {
  async function loadOlderLogs(entry: LoadMoreLogEntry) {
    if (!(messages.value[0] instanceof LoadMoreLogEntry)) throw new Error("No loadMoreLogEntry on first item");

    const [loader, ...existingLogs] = messages.value;

    if (container) {
      const to = existingLogs[0].date;
      const lastSeenId = existingLogs[0].id;
      const last = messages.value[Math.min(messages.value.length - 1, 300)].date;
      const delta = to.getTime() - last.getTime();
      const from = new Date(to.getTime() + delta);
      try {
        loadingMore.value = true;
        const { logs: newLogs, signal } = await loadBetween(container, params, from, to, {
          min: 100,
          lastSeenId,
        });
        if (newLogs && signal.aborted === false) {
          messages.value = [loader, ...newLogs, ...existingLogs];
        }
      } catch (err) {
        console.error(err);
      } finally {
        loadingMore.value = false;
      }
    } else if (containers.value.length > 0) {
      const containerIDs = new Set(containers.value.map((c) => c.id));
      const earliestByContainer = new Map<string, LogEntry<LogMessage>>();
      for (const log of existingLogs) {
        if (!log.containerID || !containerIDs.has(log.containerID)) continue;
        if (!earliestByContainer.has(log.containerID)) {
          earliestByContainer.set(log.containerID, log);
          if (earliestByContainer.size === containerIDs.size) break;
        }
      }

      const earliestDate = existingLogs[0].date;
      const latestDate = existingLogs[Math.min(existingLogs.length - 1, 300)].date;
      const delta = earliestDate.getTime() - latestDate.getTime();

      try {
        loadingMore.value = true;
        const minPerContainer = Math.ceil(100 / containers.value.length);

        const results = await Promise.all(
          containers.value.map((c) => {
            const earliest = earliestByContainer.get(c.id);
            const to = earliest?.date ?? existingLogs[0].date;
            const from = new Date(to.getTime() + delta);
            const containerRef = shallowRef(c);
            return loadBetween(containerRef, params, from, to, {
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
  }

  async function loadSkippedLogs(entry: SkippedLogsEntry) {
    const from = entry.firstSkipped.date;
    const to = entry.lastSkippedLog.date;
    const lastSeenId = entry.lastSkippedLog.id;

    if (container) {
      try {
        loadingMore.value = true;
        const { logs, signal } = await loadBetween(container, params, from, to, { lastSeenId });
        if (logs && signal.aborted === false) {
          messages.value = messages.value.slice(logs.length).flatMap((log) => (log === entry ? logs : [log]));
        }
      } catch (err) {
        console.error(err);
      } finally {
        loadingMore.value = false;
      }
    } else if (containers.value.length > 0) {
      try {
        loadingMore.value = true;
        const results = await Promise.all(
          containers.value.map((c) => {
            const containerRef = shallowRef(c);
            return loadBetween(containerRef, params, from, to, { lastSeenId });
          }),
        );
        const allLogs = results
          .filter(({ signal }) => !signal.aborted)
          .flatMap(({ logs }) => logs)
          .sort((a, b) => a.date.getTime() - b.date.getTime());

        if (allLogs.length > 0) {
          messages.value = messages.value.flatMap((log) => (log === entry ? allLogs : [log]));
        }
      } catch (err) {
        console.error(err);
      } finally {
        loadingMore.value = false;
      }
    }
  }

  return { loadOlderLogs, loadSkippedLogs };
}
