import { useCloudConfig } from "./cloudConfig";
import type { VolumeBucket } from "./searchVolume";

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
  // How every match spreads over the window; first page only, and absent when
  // Cloud is older than the field or could not count in time.
  volume?: VolumeBucket[];
  stepNs?: number;
  // Oldest timestamp (ns) the search reaches and every match in it, as Cloud
  // decided them. Absent from an older Cloud, or when it could not count.
  windowStart?: number;
  total?: number;
}

const debounceMs = 250;

/**
 * Hands a query to the Cloud results page. `available` is the one definition of
 * "Cloud can search": linked AND streaming logs, since nothing is indexed otherwise.
 */
export function useCloudSearchHandoff() {
  const { cloudConfig } = useCloudConfig();
  // Optional: a few surfaces mount without a router (tests, the login page).
  const router = useRouter();
  const available = computed(() => !!cloudConfig.value?.linked && !!cloudConfig.value?.streamLogs);

  // `regex` says the query came from find, which runs it as a regex. Cloud decides
  // what that means; Dozzle only carries the intent.
  function open(query: string, { regex = false } = {}) {
    const q = query.trim();
    if (!available.value || !q) return;
    router?.push({ path: "/cloud/search", query: regex ? { q, regex: "1" } : { q } });
  }

  return { available, open };
}

/** A search Cloud refused because of the query itself; message is user-facing. */
class QueryError extends Error {}

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
 *   504 -> timeout (3s upstream)
 *   400 / 422 -> Cloud rejected the query itself; `errorDetail` carries its
 *               message, which is written for the person who typed it
 *   any other 4xx/5xx -> error set, results cleared
 */
export function useCloudLogSearch(query: Ref<string>, regex: Ref<boolean> = ref(false)) {
  const results = ref<CloudLogHit[]>([]);
  const loading = ref(false);
  const loadingMore = ref(false);
  const error = ref<Error | null>(null);
  // Cloud's own explanation, set only when the query itself was the problem
  // (invalid, or too broad to finish). Null for every other failure, whose
  // message would describe infrastructure rather than the search.
  const errorDetail = ref<string | null>(null);
  const hasMore = ref(false);
  // Round trip of the last fresh search in ms, as the reader felt it. Cloud does
  // not report its own query time, and the wait the reader sat through is the
  // number that means something to them anyway.
  const tookMs = ref<number | null>(null);
  const volume = ref<VolumeBucket[]>([]);
  const stepNs = ref(0);
  const windowStart = ref(0);
  const total = ref<number | null>(null);
  // Cursor (timestamp_ns) of the last hit on the current page; 0 = at the
  // newest page. Cleared on every new query.
  const nextBefore = ref<number>(0);

  const { available } = useCloudSearchHandoff();

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
    errorDetail.value = null;
    loading.value = false;
    loadingMore.value = false;
    hasMore.value = false;
    nextBefore.value = 0;
    tookMs.value = null;
    volume.value = [];
    stepNs.value = 0;
    windowStart.value = 0;
    total.value = null;
  }

  async function fetchPage(q: string, before: number, signal: AbortSignal): Promise<CloudLogSearchResponse | null> {
    let url = withBase(`/api/cloud/search/logs?q=${encodeURIComponent(q)}&limit=20`);
    if (regex.value) url += "&regex=1";
    // The window's shape does not change as the reader pages back, so only the first page counts it.
    url += before > 0 ? `&before=${before}` : "&volume=1";
    const res = await fetch(url, { signal });
    if (res.status === 204) return { hits: [], hasMore: false };
    if (!res.ok) {
      if (res.status === 400 || res.status === 422) {
        const body = (await res.json().catch(() => null)) as { error?: string } | null;
        if (body?.error) throw new QueryError(body.error);
      }
      throw new Error(`cloud search failed: ${res.status}`);
    }
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
    errorDetail.value = null;
    nextBefore.value = 0;
    tookMs.value = null;

    try {
      const started = performance.now();
      const body = await fetchPage(q, 0, abortController.signal);
      if (!body) return;
      tookMs.value = Math.round(performance.now() - started);
      results.value = body.hits ?? [];
      volume.value = body.volume ?? [];
      stepNs.value = body.stepNs ?? 0;
      windowStart.value = body.windowStart ?? 0;
      total.value = body.total ?? null;
      hasMore.value = !!body.hasMore;
      nextBefore.value = body.nextBefore ?? 0;
    } catch (e) {
      if ((e as DOMException)?.name !== "AbortError") {
        error.value = e as Error;
        errorDetail.value = e instanceof QueryError ? e.message : null;
        results.value = [];
        hasMore.value = false;
        volume.value = [];
        stepNs.value = 0;
        windowStart.value = 0;
        total.value = null;
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
    [query, available, regex],
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

  return {
    results,
    loading,
    loadingMore,
    error,
    errorDetail,
    available,
    hasMore,
    tookMs,
    volume,
    stepNs,
    windowStart,
    total,
    loadMore,
  };
}
