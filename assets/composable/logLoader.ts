import { ShallowRef, type Ref } from "vue";
import { type LogMessage, LogEntry, LoadMoreLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import { loadBetween } from "@/composable/loadBetween";
import { useCloudAlerts, mergeAlerts } from "@/composable/cloudAlerts";

// Matches the rolling window size used for stats history
const LOG_WINDOW_FOR_DELTA = 300;

export function useLogLoader(
  messages: ShallowRef<LogEntry<LogMessage>[]>,
  containers: Ref<Container[]>,
  params: Ref<URLSearchParams>,
  loadingMore: Ref<boolean>,
) {
  const { fetchAlerts } = useCloudAlerts();
  // Anchor keys already placed, so overlapping scroll windows don't duplicate.
  // Keyed on (alertId, anchor) rather than alertId: one incident legitimately
  // marks every window it was active in.
  const placedAlerts = new Set<string>();
  // The viewer clears its messages when the stream changes (container switch,
  // filter change). Without this the seen-set would outlive the entries it was
  // tracking, and alerts already scrolled past would never render again.
  watch([params, containers], () => placedAlerts.clear());
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
        messages.value = [loader, ...(await withAlerts(allNewLogs)), ...existingLogs];
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

  /**
   * Decorates a freshly loaded, time-sorted run of logs with any Dozzle Cloud
   * alerts that fired inside it.
   *
   * fetchAlerts never rejects, but the guard is kept regardless: this runs on
   * the scroll path, where the log lines have already been fetched. A cloud
   * problem must degrade to "no alerts", never cost the user their logs.
   */
  async function withAlerts(logs: LogEntry<LogMessage>[]): Promise<LogEntry<LogMessage>[]> {
    if (logs.length === 0) return logs;
    try {
      const ids = containers.value.map((c) => c.id);
      const alerts = await fetchAlerts(ids, logs[0].date, new Date(logs[logs.length - 1].date.getTime() + 1));
      return mergeAlerts(logs, alerts, placedAlerts);
    } catch (err) {
      console.error(err);
      return logs;
    }
  }

  return { loadOlderLogs, loadSkippedLogs };
}
