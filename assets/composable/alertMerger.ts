import { ShallowRef, type Ref } from "vue";
import {
  type LogMessage,
  LogEntry,
  LoadMoreLogEntry,
  AlertLogEntry,
  CloudEventLogEntry,
  SkippedLogsEntry,
} from "@/models/LogEntry";
import { Container } from "@/models/Container";
import { useCloudAlerts, mergeAlerts, attachEvents, mergeCloudEvents } from "@/composable/cloudAlerts";

// Cloud aggregates events on a 15s window before an alert can even exist, so
// asking more often than that cannot surface anything sooner — it only costs
// requests. Polling on a timer also decouples the rate from log volume: this
// used to be driven by the buffer flush, which meant a chatty container hit
// Cloud roughly once a second, per open tab.
const ALERT_POLL_MS = 15_000;

/**
 * True for rows that came out of the log store, false for the ones the viewer
 * synthesises around them.
 *
 * Callers that need to anchor a range request on the edge of the list have to
 * skip the synthetic rows: an alert's id is its anchor timestamp in
 * nanoseconds, not a line hash, so handing it to the backend as `lastSeenId`
 * matches nothing and its date sits a millisecond off the line it followed.
 */
export function isStreamLog(entry: LogEntry<LogMessage>): boolean {
  return !(
    entry instanceof LoadMoreLogEntry ||
    entry instanceof AlertLogEntry ||
    entry instanceof CloudEventLogEntry ||
    entry instanceof SkippedLogsEntry
  );
}

/**
 * useAlertMerger owns the Dozzle Cloud alert layer for one log view: the dedupe
 * state, the decoration of freshly loaded windows, and the poll that catches
 * alerts firing while the view is open.
 *
 * Shared between the live stream and the historical (jump-to-date) view. Both
 * assemble their windows differently but need identical alert behaviour, and
 * keeping this in one place is what stopped the historical view from silently
 * having none.
 */
export function useAlertMerger(
  messages: ShallowRef<LogEntry<LogMessage>[]>,
  containers: Ref<Container[]>,
  params: Ref<URLSearchParams>,
) {
  const { fetchAlerts, available: alertsAvailable } = useCloudAlerts();
  // Anchor keys already placed, so overlapping scroll windows don't duplicate.
  // Keyed on (alert, anchor) rather than the alert alone: one incident legitimately
  // marks every window it was active in.
  const placedAlerts = new Set<string>();
  // Newest log timestamp the poll has already asked Cloud about. Events only
  // describe lines, so a window that gained no lines cannot have gained
  // events — and skipping them keeps a quiet container's poll from carrying a
  // payload of up to a thousand rows the viewer would discard. Alerts are
  // still requested every time: an alert's anchor is older than the alert, so
  // one can land on a line that has been on screen for a while.
  let polledThrough: number | undefined;
  // The viewer clears its messages when the stream changes (container switch,
  // filter change). Without this the seen-set would outlive the entries it was
  // tracking, and alerts already scrolled past would never render again.
  watch([params, containers], () => {
    placedAlerts.clear();
    polledThrough = undefined;
  });

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
   * Scrollback loads only run when the user scrolls, so without this an alert
   * that fired inside the window the viewer *opens* on never appeared — the
   * most common case there is, since "something just happened" is usually why
   * the container got opened at all.
   *
   * Safe to call repeatedly: placedAlerts dedupes, and an unchanged window
   * merges nothing.
   */
  async function decorateVisibleNow() {
    if (!alertsAvailable.value || containers.value.length === 0) return;

    // Load-more rows pin to the ends of the list and carry `now` as their date,
    // so they have to be held out of the merge — a time-anchored alert would
    // otherwise sort ahead of the head one and push it off the top, or past the
    // tail one in the historical view and strand it mid-list.
    const current = messages.value;
    let start = 0;
    let end = current.length;
    if (current[start] instanceof LoadMoreLogEntry) start++;
    if (end > start && current[end - 1] instanceof LoadMoreLogEntry) end--;
    const head = current.slice(0, start);
    const tail = current.slice(end);
    const logs = current.slice(start, end);
    if (logs.length === 0) return;

    try {
      const from = logs[0].date;
      const to = new Date(logs[logs.length - 1].date.getTime() + 1);
      // Origins only, like scrollback. An incident already running when this
      // window opens shows through the per-line badges instead, which is both
      // more precise and cheaper than a second block.
      const newest = logs[logs.length - 1].date.getTime();
      const wantEvents = polledThrough === undefined || newest > polledThrough;
      polledThrough = newest;

      const { alerts, events } = await fetchAlerts(
        containers.value.map((c) => c.id),
        from,
        to,
        { events: wantEvents },
      );

      // Badges mutate the entries in place, so the list has to be reassigned
      // for Vue to see it — messages is a shallowRef.
      const badged = attachEvents(logs, events);
      const withEvents = mergeCloudEvents(logs, events, placedAlerts);
      const merged = mergeAlerts(withEvents, alerts, placedAlerts);
      if (!badged && merged === logs) return;
      messages.value = [...head, ...merged, ...tail];
    } catch (err) {
      console.error(err);
    }
  }

  // The opening window is assembled without alerts — only scrollback fetches
  // them — and the frames that assemble it arrive as a burst describing the
  // same window, so callers get a debounced entry point rather than the raw one.
  const decorateVisible = useDebounceFn(decorateVisibleNow, 400);

  // An alert's anchor is the timestamp of the event that triggered it, which is
  // always older than the alert itself — so the poll re-asks the whole visible
  // window rather than only the slice since the last call. A window that only
  // moved forward would never see an alert land on a line already on screen.
  // This matters in the historical view too, where the window never moves at
  // all: triage can attach an alert to a line that scrolled past days ago.
  const visible = useDocumentVisibility();
  const alertPoll = useIntervalFn(
    () => {
      if (visible.value === "visible") decorateVisibleNow();
    },
    ALERT_POLL_MS,
    { immediate: false },
  );

  // Only runs when Cloud is actually linked. An unlinked Dozzle has no alerts
  // to fetch, so a timer there is pure waste — and an unconditional interval
  // also hangs any test that drains pending timers.
  //
  // The immediate pass on becoming linked matters twice. The cloud config is
  // fetched asynchronously at boot, so on an ordinary load this flips true
  // *after* the view has already assembled its first window — without it the
  // opening window would sit bare until the first tick. It also covers linking
  // mid-session: alerts start appearing at once rather than up to a poll later.
  watch(
    alertsAvailable,
    (linked) => {
      if (!linked) {
        alertPoll.pause();
        return;
      }
      alertPoll.resume();
      decorateVisible();
    },
    { immediate: true },
  );

  return { withAlerts, decorateVisible, alertsAvailable };
}
