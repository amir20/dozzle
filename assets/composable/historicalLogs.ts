import { HistoricalContainer } from "@/models/Container";
import { LogMessage, LoadMoreLogEntry, LogEntry } from "@/models/LogEntry";
import { ShallowRef } from "vue";
import { loadBetween } from "@/composable/loadBetween";
import { useAlertMerger, isStreamLog } from "@/composable/alertMerger";

export function useHistoricalContainerLog(historicalContainer: Ref<HistoricalContainer>): LogStreamSource {
  const messages: ShallowRef<LogEntry<LogMessage>[]> = shallowRef([]);
  const opened = ref(false);
  const loading = ref(true);
  const error = ref(false);
  // Historical views are a fixed window around a log id, never a running search.
  const searchStatus = ref<SearchStatus>({ active: false, done: false, matches: 0 });
  const container = toRef(() => historicalContainer.value.container);

  const { streamConfig, levels, loadingMore } = useLoggingContext();
  const { isSearching, debouncedSearchFilter, inverseFilter } = useSearchFilter();

  const params = computed(() => {
    const params = new URLSearchParams();
    if (streamConfig.value.stdout) params.append("stdout", "1");
    if (streamConfig.value.stderr) params.append("stderr", "1");
    if (isSearching.value) {
      params.append("filter", debouncedSearchFilter.value);
      if (inverseFilter.value) params.append("inverse", "true");
    }
    for (const level of levels.value) {
      params.append("levels", level);
    }
    return params;
  });

  // Same alert layer the live stream uses: without it a deep link to an alert
  // landed on the one view that could not draw it.
  const containers = computed(() => [container.value]);
  const { withAlerts, decorateVisible } = useAlertMerger(messages, containers, params);

  const route = useRoute();
  async function loadLogs() {
    loadingMore.value = true;
    try {
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
      messages.value = [loaderOlder, ...(await withAlerts([...before, ...after])), loadNewer];
      loading.value = false;
      opened.value = true;
      // Covers the boot race: the cloud config arrives asynchronously, so
      // withAlerts above may have run while the instance still looked unlinked.
      decorateVisible();
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
      if (!item) return;
      const { logs, signal } = await loadBetween(container, params, item.date, new Date(), {
        maxStart: 100,
        startId: item.id,
      });

      if (signal.aborted) {
        return;
      }

      if (!logs.length) {
        return;
      }

      const loader = messages.value.at(-1)!;
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
