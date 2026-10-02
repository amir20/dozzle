import { HistoricalContainer } from "@/models/Container";
import { LogMessage, LoadMoreLogEntry, LogEntry, RangeEdgeLogEntry } from "@/models/LogEntry";
import { ShallowRef } from "vue";
import { loadBetween } from "./loadBetween";
import { FETCH_PAGE } from "./logLoader";
import { timeRangeRoute } from "./timeRange";
import { useAlertMerger, isStreamLog } from "@/composable/cloud/alertMerger";

export function useHistoricalContainerLog(historicalContainer: Ref<HistoricalContainer>): LogStreamSource {
  const { t } = useI18n();
  const messages: ShallowRef<LogEntry<LogMessage>[]> = shallowRef([]);
  const opened = ref(false);
  const loading = ref(true);
  const error = ref(false);
  // Historical views are a fixed window around a log id, never a running search.
  const searchStatus = ref<SearchStatus>({ active: false, done: false, matches: 0 });
  const container = toRef(() => historicalContainer.value.container);

  const { streamConfig, levels, loadingMore } = useLoggingContext();
  const { isSearching, appliedSearchFilter, inverseFilter } = useSearchFilter();

  const params = computed(() => {
    const params = new URLSearchParams();
    if (streamConfig.value.stdout) params.append("stdout", "1");
    if (streamConfig.value.stderr) params.append("stderr", "1");
    if (isSearching.value) {
      params.append("filter", appliedSearchFilter.value);
      if (inverseFilter.value) params.append("inverse", "true");
    }
    for (const level of levels.value) {
      params.append("levels", level);
    }
    return params;
  });

  // Same alert layer the live stream uses: without it a deep link to an alert
  // landed on the one view that could not draw it.
  //
  // The date is handed over as the anchor. A metric or container alert has no
  // log line, so the moment this view was opened on can sit outside the range
  // of every line loaded around it — on a quiet container that meant clicking
  // "show me the logs around it" for a CPU spike landed on a window that never
  // asked Cloud about the spike.
  const containers = computed(() => [container.value]);
  const anchor = () => historicalContainer.value.date;
  const { withAlerts, decorateVisible, alertsAvailable } = useAlertMerger(messages, containers, params, anchor);

  const route = useRoute();
  const router = useRouter();
  const until = computed(() => historicalContainer.value.until);

  // The ends of a range. Moving the start opens a new window, so it navigates;
  // moving the end only lets the bottom keep loading, so the window stays.
  function startEdge() {
    const { date, until: end } = historicalContainer.value;
    const earlier = (ms: number) => () =>
      router.replace(
        timeRangeRoute(container.value.id, { kind: "range", from: new Date(date.getTime() - ms), until: end! }),
      );
    return new RangeEdgeLogEntry(date, "start", [
      { label: "1m", run: earlier(60_000) },
      { label: "5m", run: earlier(5 * 60_000) },
      { label: "15m", run: earlier(15 * 60_000) },
    ]);
  }

  function endEdge() {
    const { date, until: end } = historicalContainer.value;
    const later = (ms: number) => () =>
      router.replace(
        timeRangeRoute(container.value.id, { kind: "range", from: date, until: new Date(end!.getTime() + ms) }),
      );
    return new RangeEdgeLogEntry(end!, "end", [
      { label: "1m", run: later(60_000) },
      { label: "5m", run: later(5 * 60_000) },
      {
        label: t("time-range.to-now"),
        run: () => router.push(timeRangeRoute(container.value.id, { kind: "since", since: date })),
      },
    ]);
  }

  // A range reads forward from its start, a page at a time, and each page is a
  // read of the log from its beginning on Docker's side: pages are as large as
  // a fetch allows so a range takes as few of them as it can.
  async function loadRange(end: Date) {
    const { logs } = await loadBetween(container, params, historicalContainer.value.date, end, {
      maxStart: FETCH_PAGE,
    });
    const bottom = logs.length < FETCH_PAGE ? endEdge() : new LoadMoreLogEntry(new Date(), loadNewerLogs, false);
    messages.value = [startEdge(), ...(await withAlerts(logs)), bottom];
  }

  // A later end reopens the bottom of a window that had reached the old one.
  watch(until, (end, previous) => {
    if (!end || !previous || end <= previous) return;
    const last = messages.value.at(-1);
    if (last instanceof RangeEdgeLogEntry && last.edge === "end") {
      messages.value = [...messages.value.slice(0, -1), new LoadMoreLogEntry(new Date(), loadNewerLogs, false)];
    }
  });

  async function loadLogs() {
    loadingMore.value = true;
    try {
      if (until.value) {
        const wasLinked = alertsAvailable.value;
        await loadRange(until.value);
        loading.value = false;
        opened.value = true;
        if (!wasLinked && alertsAvailable.value) decorateVisible();
        return;
      }
      const lastSeenId = route.query.logId ? +route.query.logId : undefined;
      const [{ logs: before }, { logs: after }] = await Promise.all([
        loadBetween(
          container,
          params,
          new Date(historicalContainer.value.date.getTime() - 1000 * 60 * 5),
          new Date(historicalContainer.value.date.getTime() + 1000),
          {
            min: 50,
            lastSeenId,
          },
        ),
        loadBetween(container, params, historicalContainer.value.date, new Date(), {
          maxStart: 50,
        }),
      ]);
      const loaderOlder = new LoadMoreLogEntry(new Date(), loadOlderLogs);
      const loadNewer = new LoadMoreLogEntry(new Date(), loadNewerLogs, false);
      const wasLinked = alertsAvailable.value;
      messages.value = [loaderOlder, ...(await withAlerts([...before, ...after])), loadNewer];
      loading.value = false;
      opened.value = true;
      // Only for the boot race: the cloud config arrives asynchronously, so
      // withAlerts above may have run while the instance still looked unlinked.
      // When it was already linked the window is decorated, and asking again
      // would just spend a request on a window that cannot have changed.
      if (!wasLinked && alertsAvailable.value) decorateVisible();
    } catch (error) {
      console.error(error);
    } finally {
      loadingMore.value = false;
    }
  }

  watchArray([params, container], loadLogs, { immediate: true });

  async function loadOlderLogs(entry: LoadMoreLogEntry) {
    loadingMore.value = true;
    try {
      // First real line, not messages[1]: alert and event rows sit between the
      // loader and the logs, and their id is an anchor timestamp rather than a
      // line hash, so using one as lastSeenId would match nothing.
      const item = messages.value.find(isStreamLog);
      if (!item) return;
      const { logs, signal } = await loadBetween(
        container,
        params,
        new Date(item.date.getTime() - 1000 * 60 * 5),
        item.date,
        {
          min: 200,
          lastSeenId: item.id,
        },
      );

      if (signal.aborted) {
        return;
      }

      if (!logs.length) {
        return;
      }

      const [loader, ...rest] = messages.value;
      messages.value = [loader, ...(await withAlerts(logs)), ...rest];
    } catch (error) {
      console.error(error);
    } finally {
      loadingMore.value = false;
    }
  }

  async function loadNewerLogs(entry: LoadMoreLogEntry) {
    loadingMore.value = true;
    try {
      // Last real line, for the same reason loadOlderLogs skips back past the
      // synthetic rows.
      const item = messages.value.findLast(isStreamLog);
      const end = until.value;
      // An empty range has no line to continue from, only its start.
      if (!item && !end) return;
      const { logs, signal } = await loadBetween(
        container,
        params,
        item?.date ?? historicalContainer.value.date,
        end ?? new Date(),
        {
          maxStart: end ? FETCH_PAGE : 100,
          startId: item?.id,
        },
      );

      if (signal.aborted) {
        return;
      }

      // Inside a range, a short page means the end is reached.
      const reachedEnd = end !== undefined && logs.length < FETCH_PAGE;
      if (!logs.length && !reachedEnd) {
        return;
      }

      const loader = reachedEnd ? endEdge() : messages.value.at(-1)!;
      const rest = messages.value.slice(0, -1);
      messages.value = [...rest, ...(await withAlerts(logs)), loader];
    } catch (error) {
      console.error(error);
    } finally {
      loadingMore.value = false;
    }
  }

  return {
    messages,
    opened,
    error,
    loading,
    searchStatus,
  };
}
