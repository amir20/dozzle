import { ShallowRef, type Ref } from "vue";
import { type LogMessage, LogEntry, LoadMoreLogEntry, SkippedLogsEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import { loadBetween } from "@/composable/loadBetween";
import { useCloudAlerts, mergeAlerts, attachEvents, mergeCloudEvents } from "@/composable/cloudAlerts";

// Matches the rolling window size used for stats history
const LOG_WINDOW_FOR_DELTA = 300;

export function useLogLoader(
  messages: ShallowRef<LogEntry<LogMessage>[]>,
  containers: Ref<Container[]>,
  params: Ref<URLSearchParams>,
  loadingMore: Ref<boolean>,
) {
  const { fetchAlerts, available: alertsAvailable } = useCloudAlerts();
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
    if (!alertsAvailable.value || logs.length === 0) return logs;
    try {
      const ids = containers.value.map((c) => c.id);
      const { alerts, events } = await fetchAlerts(
        ids,
        logs[0].date,
        new Date(logs[logs.length - 1].date.getTime() + 1),
        { events: true },
      );
      attachEvents(logs, events);
      return mergeAlerts(mergeCloudEvents(logs, events, placedAlerts), alerts, placedAlerts);
    } catch (err) {
      console.error(err);
      return logs;
    }
  }

  /**
   * Decorates the logs already on screen with any alerts that fired inside
   * them.
   *
   * loadOlderLogs only runs when the user scrolls back, so without this an
   * alert that fired inside the window the viewer *opens* on never appeared —
   * the most common case there is, since "something just happened" is usually
   * why the container got opened at all.
   *
   * Safe to call repeatedly: placedAlerts dedupes, and an unchanged window
   * merges nothing.
   */
  async function loadAlertsForVisible() {
    if (!alertsAvailable.value || containers.value.length === 0) return;

    // The loader pins to the top of the list and carries `now` as its date, so
    // it must be held out of the merge — a time-anchored alert would otherwise
    // sort ahead of it and push it off the top.
    const [head, ...rest] = messages.value;
    const loader = head instanceof LoadMoreLogEntry ? head : undefined;
    const logs = loader ? rest : messages.value;
    if (logs.length === 0) return;

    try {
      const from = logs[0].date;
      const to = new Date(logs[logs.length - 1].date.getTime() + 1);
      // Origins only, like scrollback. An incident already running when this
      // window opens shows through the per-line badges instead, which is both
      // more precise and cheaper than a second block.
      const { alerts, events } = await fetchAlerts(
        containers.value.map((c) => c.id),
        from,
        to,
        { events: true },
      );

      // Badges mutate the entries in place, so the list has to be reassigned
      // for Vue to see it — messages is a shallowRef.
      const badged = attachEvents(logs, events);
      const withEvents = mergeCloudEvents(logs, events, placedAlerts);
      const merged = mergeAlerts(withEvents, alerts, placedAlerts);
      if (!badged && merged === logs) return;
      messages.value = loader ? [loader, ...merged] : merged;
    } catch (err) {
      console.error(err);
    }
  }

  return { loadOlderLogs, loadSkippedLogs, loadAlertsForVisible, alertsAvailable };
}
