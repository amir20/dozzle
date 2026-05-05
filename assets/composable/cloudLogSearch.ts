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
  // Cursor for the next older page. Pass back as `before=` in the URL.
  // Omitted when there's nothing more to load.
  nextBefore?: number;
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
  const loadingMore = ref(false);
  const error = ref<Error | null>(null);
  const hasMore = ref(false);
  // Cursor (timestamp_ns) of the last hit on the current page; 0 = at the
  // newest page. Cleared on every new query.
  const nextBefore = ref<number>(0);

  const available = computed(() => !!cloudConfig.value?.linked && !!cloudConfig.value?.streamLogs);

  // Two parallel fetch lifecycles — keystroke search (cancels on next
  // keystroke / unmount) and pagination loadMore (cancels on unmount or
  // when a new query lands and supersedes pagination state). Tracking
  // them separately avoids the keystroke aborter cancelling an in-flight
  // pagination request and vice versa.
  let abortController: AbortController | null = null;
  let loadMoreAborter: AbortController | null = null;

  function clearResults() {
    results.value = [];
    error.value = null;
    loading.value = false;
    loadingMore.value = false;
    hasMore.value = false;
    nextBefore.value = 0;
  }

  async function fetchPage(q: string, before: number, signal: AbortSignal): Promise<CloudLogSearchResponse | null> {
    let url = withBase(`/api/cloud/search/logs?q=${encodeURIComponent(q)}&limit=20`);
    if (before > 0) url += `&before=${before}`;
    const res = await fetch(url, { signal });
    if (res.status === 204) return { hits: [], hasMore: false };
    if (!res.ok) throw new Error(`cloud search failed: ${res.status}`);
    return (await res.json()) as CloudLogSearchResponse;
  }

  async function runSearch(q: string) {
    if (abortController) abortController.abort();
    // A fresh query supersedes any in-flight pagination — that page is
    // for the previous query and would be appended onto the wrong result
    // set if it landed late.
    loadMoreAborter?.abort();
    abortController = new AbortController();
    loading.value = true;
    error.value = null;
    nextBefore.value = 0;

    try {
      const body = await fetchPage(q, 0, abortController.signal);
      if (!body) return;
      results.value = body.hits ?? [];
      hasMore.value = !!body.hasMore;
      nextBefore.value = body.nextBefore ?? 0;
    } catch (e) {
      if ((e as DOMException)?.name !== "AbortError") {
        error.value = e as Error;
        results.value = [];
        hasMore.value = false;
      }
    } finally {
      loading.value = false;
    }
  }

  // loadMore appends the next older page. Safe to call repeatedly — guarded
  // by hasMore + a separate loading flag so the input-debounced search and
  // the user-triggered pagination don't trip each other.
  async function loadMore() {
    if (loadingMore.value || !hasMore.value || nextBefore.value <= 0) return;
    const q = query.value.trim();
    if (!q) return;
    loadingMore.value = true;
    loadMoreAborter?.abort();
    loadMoreAborter = new AbortController();
    try {
      const body = await fetchPage(q, nextBefore.value, loadMoreAborter.signal);
      if (!body) return;
      results.value = [...results.value, ...(body.hits ?? [])];
      hasMore.value = !!body.hasMore;
      nextBefore.value = body.nextBefore ?? 0;
    } catch (e) {
      if ((e as DOMException)?.name !== "AbortError") error.value = e as Error;
    } finally {
      loadingMore.value = false;
    }
  }

  watchDebounced(
    [query, available],
    ([q, isAvailable]) => {
      const trimmed = q.trim();
      if (!isAvailable || trimmed === "") {
        clearResults();
        return;
      }
      runSearch(trimmed);
    },
    { debounce: debounceMs, immediate: true },
  );

  onScopeDispose(() => {
    abortController?.abort();
    loadMoreAborter?.abort();
  });

  return { results, loading, loadingMore, error, available, hasMore, loadMore };
}
