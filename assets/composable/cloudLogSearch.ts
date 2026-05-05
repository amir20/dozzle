import { useCloudConfig } from "@/composable/cloudConfig";

export interface CloudLogHit {
  ts: number;
  hostId: string;
  containerId: string;
  containerName: string;
  message: string;
  stream: string;
  level: string;
  // Dozzle's deterministic FNV-32a id for the raw log line — used to deep-link
  // to the exact line in the local log viewer. Optional: pre-indexing logs
  // (or older Dozzle clients) won't have it.
  logId?: number;
}

interface CloudLogSearchResponse {
  hits: CloudLogHit[];
  hasMore: boolean;
}

const debounceMs = 250;

/**
 * useCloudLogSearch performs Cloud-side log search via the Dozzle backend's
 * /api/cloud/search/logs endpoint. Identity is derived server-side from the
 * authenticated gRPC connection; this composable passes only the query.
 *
 * Behavior:
 *   - debounced 250ms; whitespace-only short-circuits to []
 *   - aborts any in-flight request on each new keystroke (AbortController)
 *   - `available` is computed: cloud linked AND streamLogs enabled
 *   - when `available` is false, results stay [] regardless of query
 *
 * Status mapping:
 *   200 -> hits populated (may be empty)
 *   204 -> streaming disabled server-side (defense-in-depth)
 *   503 -> cloud not configured
 *   504 -> timeout (500ms upstream)
 *   any other 4xx/5xx -> error set, results cleared
 */
export function useCloudLogSearch(query: Ref<string>) {
  const { cloudConfig } = useCloudConfig();

  const results = ref<CloudLogHit[]>([]);
  const loading = ref(false);
  const error = ref<Error | null>(null);

  const available = computed(() => !!cloudConfig.value?.linked && !!cloudConfig.value?.streamLogs);

  let abortController: AbortController | null = null;
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  function clearTimer() {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
  }

  function clearResults() {
    results.value = [];
    error.value = null;
    loading.value = false;
  }

  async function runSearch(q: string) {
    if (abortController) abortController.abort();
    abortController = new AbortController();
    loading.value = true;
    error.value = null;

    try {
      const url = withBase(`/api/cloud/search/logs?q=${encodeURIComponent(q)}&limit=20`);
      const res = await fetch(url, { signal: abortController.signal });

      if (res.status === 204) {
        // streamLogs disabled server-side — silent
        results.value = [];
        return;
      }
      if (!res.ok) {
        error.value = new Error(`cloud search failed: ${res.status}`);
        results.value = [];
        return;
      }

      const body = (await res.json()) as CloudLogSearchResponse;
      results.value = body.hits ?? [];
    } catch (e) {
      // AbortError is normal — ignore
      if ((e as DOMException)?.name !== "AbortError") {
        error.value = e as Error;
        results.value = [];
      }
    } finally {
      loading.value = false;
    }
  }

  watch(
    [query, available],
    ([q, isAvailable]) => {
      clearTimer();
      const trimmed = q.trim();
      if (!isAvailable || trimmed === "") {
        clearResults();
        return;
      }
      debounceTimer = setTimeout(() => runSearch(trimmed), debounceMs);
    },
    { immediate: true },
  );

  onScopeDispose(() => {
    clearTimer();
    abortController?.abort();
  });

  return { results, loading, error, available };
}
