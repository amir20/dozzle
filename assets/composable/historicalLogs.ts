import { HistoricalContainer } from "@/models/Container";
import { JSONObject, LogEntry } from "@/models/LogEntry";
import { ShallowRef } from "vue";
import { loadBetween } from "@/composable/eventStreams";

export function useHistoricalContainerLog(historicalContainer: Ref<HistoricalContainer>): LogStreamSource {
  const url = computed(
    () =>
      `/api/hosts/${historicalContainer.value.container.host}/containers/${historicalContainer.value.container.id}/logs`,
  );

  const messages: ShallowRef<LogEntry<string | JSONObject>[]> = shallowRef([]);
  const opened = ref(false);
  const loading = ref(true);
  const error = ref(false);
  const { streamConfig, levels, loadingMore } = useLoggingContext();

  const params = computed(() => {
    const params = new URLSearchParams();
    if (streamConfig.value.stdout) params.append("stdout", "1");
    if (streamConfig.value.stderr) params.append("stderr", "1");
    // if (isSearching.value) params.append("filter", debouncedSearchFilter.value);
    for (const level of levels.value) {
      params.append("levels", level);
    }
    return params;
  });

  async function loadLogs() {
    loadingMore.value = true;
    try {
      const [{ logs: before }, { logs: after }] = await Promise.all([
        loadBetween(
          url,
          params,
          new Date(historicalContainer.value.date.getTime() - 1000 * 60 * 5),
          historicalContainer.value.date,
          {
            min: 10,
            maxEnd: 10,
          },
        ),
        loadBetween(url, params, historicalContainer.value.date, new Date(), {
          maxStart: 10,
        }),
      ]);
      messages.value = [...before, ...after];
      loading.value = false;
      opened.value = true;
    } catch (error) {
      console.error(error);
    } finally {
      loadingMore.value = false;
    }
  }

  loadLogs();

  return {
    messages,
    opened,
    error,
    loading,
    eventSourceURL: url,
  };
}
