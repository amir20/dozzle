import type { Container } from "@/models/Container";

export type HistogramWindow = { from: Date; to: Date };

export type LogHistogram = {
  start: Date;
  /** Bucket width in milliseconds. */
  width: number;
  total: number[];
  errors: number[];
  /** Set when the count stopped short: buckets that end before it are unknown, not empty. */
  scannedFrom?: Date;
  /** Only for a window with no lines: the newest line before it and the oldest after. */
  before?: Date;
  after?: Date;
};

type Response = {
  start: string;
  width: number;
  total: number[];
  errors: number[];
  scannedFrom?: string;
  before?: string;
  after?: string;
};

const optionalDate = (value?: string) => (value ? new Date(value) : undefined);

export const isEmptyHistogram = (h: LogHistogram) => h.total.every((n) => n === 0);

/**
 * Lines per bucket across a window, counted next to the log by the server.
 *
 * Asks again only when the window changes, a little after it settles, so a
 * drag on the chart does not send a request per pixel. A newer request aborts
 * the one in flight.
 */
export function useLogHistogram(
  container: Ref<Container>,
  window: Ref<HistogramWindow | undefined>,
  buckets: MaybeRefOrGetter<number> = 60,
) {
  const data = shallowRef<LogHistogram>();
  const loading = ref(false);
  // An agent older than the server cannot count; the chart is then left out.
  const unsupported = ref(false);
  let controller: AbortController | undefined;

  async function load() {
    controller?.abort();
    const w = window.value;
    if (!w || unsupported.value) return;
    controller = new AbortController();
    const signal = controller.signal;
    const params = new URLSearchParams({
      from: w.from.toISOString(),
      to: w.to.toISOString(),
      buckets: String(toValue(buckets)),
    });
    const { host, id } = container.value;
    loading.value = true;
    try {
      const response = await fetch(withBase(`/api/hosts/${host}/containers/${id}/logs/histogram?${params}`), {
        signal,
      });
      if (response.status === 501) {
        unsupported.value = true;
        data.value = undefined;
        return;
      }
      if (!response.ok) throw new Error(await response.text());
      const body = (await response.json()) as Response;
      data.value = {
        start: new Date(body.start),
        width: body.width * 1000,
        total: body.total,
        errors: body.errors,
        scannedFrom: optionalDate(body.scannedFrom),
        before: optionalDate(body.before),
        after: optionalDate(body.after),
      };
    } catch (e) {
      if (!signal.aborted) console.error(e);
    } finally {
      if (!signal.aborted) loading.value = false;
    }
  }

  const debounced = useDebounceFn(load, 150);
  watch(
    () => [window.value?.from.getTime(), window.value?.to.getTime(), container.value.id, toValue(buckets)],
    () => debounced(),
    { immediate: true },
  );
  onScopeDispose(() => controller?.abort());

  return { data, loading, unsupported };
}
