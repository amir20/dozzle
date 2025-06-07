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
    const { logs: newLogs, signal } = await loadBetween(
      url,
      loadingMore,
      params,
      historicalContainer.value.date,
      new Date(),
      0,
      300,
    );
    messages.value = newLogs;
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
