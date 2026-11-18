<template>
  <!--
    The Metrics tab on a page about the whole instance: each host's containers
    summed, read back from what this instance has been pushing to Cloud. The
    live numbers are already on the home page; this is the history behind them.
  -->
  <div ref="root" class="h-full [scrollbar-gutter:stable] overflow-y-auto p-4">
    <div class="mb-4 flex items-center gap-2">
      <span class="text-base-content/60 truncate text-xs font-semibold tracking-wide uppercase">
        {{ $t("label.host-count", cloudHasData ? hostSeries.length : live.length) }}
      </span>
      <div role="tablist" class="tabs tabs-box tabs-xs ml-auto shrink-0">
        <button
          v-for="option in windows"
          :key="option.value"
          role="tab"
          class="tab"
          :class="{ 'tab-active': window === option.value }"
          @click="window = option.value"
        >
          {{ option.label }}
        </button>
      </div>
    </div>

    <p v-if="loading" class="text-base-content/60 text-sm">{{ $t("cloud-rail.metrics-loading") }}</p>
    <!-- Too little in Cloud to draw: each host's live totals from this browser,
         until Cloud's history fills in. -->
    <template v-else-if="!cloudHasData">
      <p v-if="failed" class="text-base-content/60 mb-4 text-sm">{{ $t("cloud-rail.metrics-failed") }}</p>
      <RailLiveMetrics :series="live" named />
    </template>

    <template v-else>
      <!-- One pointer per host: hovering any of its charts marks the same
           moment on both, as in the container history. Leaving falls back to
           each chart's peak for the window. -->
      <section v-for="host in hostSeries" :key="host.id" class="mb-6 last:mb-0" :data-host="host.id">
        <h3 class="mb-2 truncate font-mono text-sm font-semibold">{{ host.name }}</h3>
        <div v-for="chart in host.charts" :key="chart.key" class="mb-3 last:mb-0">
          <div class="mb-1 flex items-baseline justify-between gap-2">
            <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ chart.label }}</span>
            <span class="truncate font-mono text-xs tabular-nums">
              <span class="font-semibold">{{ readout(host, chart).main }}</span>
              <span class="text-base-content/40">{{ readout(host, chart).aside }}</span>
            </span>
          </div>
          <p class="text-base-content/40 mb-1 truncate font-mono text-xs tabular-nums" data-summary>
            <span class="text-base-content/60 font-semibold">{{ chart.summary }}</span>{{
              chart.limit ? ` / ${chart.limit}` : ""
            }}
          </p>
          <BarChart
            :chart-data="chart.data"
            :bar-class="chart.colorClass"
            :hover-index="hovered?.id === host.id ? hovered.index : null"
            :max="chart.ceiling"
            class="h-12"
            @hover-value="(_: number, index: number, bars: number) => (hovered = { id: host.id, index, bars })"
            @hover-end="hovered = null"
          />
        </div>

        <!-- Where the window starts and ends, so its span reads without a hover. -->
        <div
          v-if="host.points.length"
          class="text-base-content/40 mt-2 flex justify-between font-mono text-xs tabular-nums"
          data-range
        >
          <span>{{ clock.format(new Date(host.range.from)) }}</span>
          <span>{{ clock.format(new Date(host.range.to)) }}</span>
        </div>

        <!-- The hovered moment is named once per host, under its charts. -->
        <p class="text-base-content/40 mt-2 truncate font-mono text-xs tabular-nums" data-moment>
          <template v-if="hovered?.id === host.id">{{ timeOf(host, hovered) }} · </template>
          {{ $t("cloud-rail.metrics-bucket", { duration: bucketLabel }) }}
        </p>
      </section>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { BarDataPoint } from "@/components/ui/BarChart.vue";
import type { Container } from "@/models/Container";
import { hostLiveSeries, MIN_CLOUD_POINTS } from "@/composable/cloud/liveMetrics";
import { hoverSpan, spanMiddle, useMetricsClock, windowClock, type MetricsHover } from "@/composable/cloud/metricsTime";

type HostPoint = { ts: number; cpu: number; memoryUsage: number };
type HostSeries = { hostId: string; points: HostPoint[] };

const { t } = useI18n();
const { hosts } = useHosts();
const store = useContainerStore();
const { metricsLookbackHours, ensureCloudStatus } = useCloudConfig();
onMounted(ensureCloudStatus);

// The same windows the container history offers, so switching pages keeps the
// question the same.
const windows = computed(() => [
  { value: "1h", label: t("cloud-rail.window-1h") },
  { value: "6h", label: t("cloud-rail.window-6h") },
  { value: "24h", label: t("cloud-rail.window-24h") },
  ...(metricsLookbackHours.value >= 168 ? [{ value: "168h", label: t("cloud-rail.window-7d") }] : []),
]);
const window = ref("24h");
watch(windows, (options) => {
  if (!options.some((o) => o.value === window.value)) window.value = "24h";
});

const series = ref<HostSeries[]>([]);
const bucket = ref(0);
const loading = ref(false);
const failed = ref(false);

// One bucket per chart column, as in the container history.
const root = ref<HTMLElement | null>(null);
const { width: rootWidth } = useElementSize(root);
const columns = ref(0);
const measure = () => (columns.value = Math.min(1000, Math.max(1, Math.floor(rootWidth.value / 5))));
watchDebounced(rootWidth, (w) => w > 0 && measure(), { debounce: 300 });
watch(rootWidth, (w) => w > 0 && columns.value === 0 && measure(), { immediate: true });

// Only the newest request may write; see RailMetrics.
let latest = 0;

async function load() {
  if (columns.value === 0) return;
  const request = ++latest;
  const current = () => request === latest;
  loading.value = true;
  failed.value = false;
  try {
    const res = await fetch(withBase(`/api/cloud/metrics/hosts?window=${window.value}&buckets=${columns.value}`));
    if (!current()) return;
    if (!res.ok) {
      // 503 is an unlinked instance, which the drawer is not offered on anyway.
      failed.value = res.status !== 503;
      series.value = [];
      return;
    }
    const body = await res.json();
    if (!current()) return;
    series.value = body.hosts ?? [];
    bucket.value = body.bucket ?? 0;
  } catch {
    if (!current()) return;
    failed.value = true;
    series.value = [];
  } finally {
    if (current()) loading.value = false;
  }
}

watch([window, columns], load, { immediate: true });

const cloudHasData = computed(() => series.value.some((s) => s.points.length >= MIN_CLOUD_POINTS));
// Every host this instance reaches, summed from the containers' live samples.
const live = computed(() =>
  Object.values(hosts.value)
    .filter((h) => h.available)
    .map((h) => hostLiveSeries(h, store.containers as Container[])),
);

const pct = (v: number) => `${v.toFixed(1)}%`;
const bytes = (v: number) => formatBytes(v, { short: true, decimals: 1 });
const mean = (xs: number[], [from, to]: [number, number]) =>
  xs.slice(from, to).reduce((a, b) => a + b, 0) / Math.max(1, to - from);

type Readout = { main: string; aside: string };
type Chart = {
  key: string;
  label: string;
  colorClass: string;
  data: BarDataPoint[];
  ceiling?: number;
  summary: string;
  limit?: string;
  // What the header says with the pointer elsewhere, and on a hovered bar.
  rest: Readout;
  at: (span: [number, number]) => Readout;
};
type Host = { id: string; name: string; points: HostPoint[]; charts: Chart[]; range: { from: number; to: number } };

// The bar under the pointer, and the host whose charts it is on. Hosts are not
// guaranteed the same number of points, so one host's index is not another's moment.
const hovered = ref<(MetricsHover & { id: string }) | null>(null);
watch(window, () => (hovered.value = null));

// A host Cloud remembers but Dozzle no longer connects to keeps its id as its
// name: its history is still real, and the id is what it was known by.
const hostSeries = computed<Host[]>(() =>
  series.value.map(({ hostId, points }) => {
    const host = hosts.value[hostId];
    const cpu = points.map((p) => p.cpu);
    const memory = points.map((p) => p.memoryUsage);
    const memTotal = host?.memTotal ?? 0;
    const peakWord = t("cloud-rail.metrics-peak");
    const whole: [number, number] = [0, points.length];
    const charts: Chart[] = [
      // The sum of each container's CPU, which reads against the host's cores
      // for a container with no limit. Scaled to its own peak: a container with
      // a limit reports against that, so the sum can pass 100.
      {
        key: "cpu",
        label: t("label.cpu"),
        colorClass: "text-primary",
        data: cpu.map((v) => ({ percent: v, value: v })),
        summary: `${t("analytics.avg")} ${pct(mean(cpu, whole))}`,
        rest: { main: pct(Math.max(0, ...cpu)), aside: ` / ${peakWord}` },
        // Cloud sends host sums without a max, so a bar reads as its average.
        at: (span) => ({ main: pct(mean(cpu, span)), aside: "" }),
      },
      // Drawn against the host's memory when Dozzle knows it, like the host card.
      {
        key: "memory",
        label: t("label.mem"),
        colorClass: "text-secondary",
        data: memory.map((v) => ({ percent: memTotal ? (v / memTotal) * 100 : v, value: v })),
        ceiling: memTotal ? 100 : undefined,
        summary: `${t("analytics.avg")} ${bytes(mean(memory, whole))}`,
        limit: memTotal ? bytes(memTotal) : undefined,
        rest: { main: bytes(Math.max(0, ...memory)), aside: ` / ${peakWord}` },
        at: (span) => ({ main: bytes(mean(memory, span)), aside: "" }),
      },
    ];
    // The time the bars cover: from the first bucket's start to the last one's end.
    const first = points[0];
    const last = points.at(-1);
    const range = first && last ? { from: first.ts, to: last.ts + bucket.value } : { from: 0, to: 0 };
    return { id: hostId, name: host?.name ?? hostId, points, charts, range };
  }),
);

const readout = (host: Host, chart: Chart) =>
  hovered.value?.id === host.id ? chart.at(hoverSpan(host.points.length, hovered.value)) : chart.rest;

const clock = useMetricsClock(() => windowClock(window.value));

// A bar covers a run of points; its middle is the honest moment to name.
function timeOf(host: Host, hover: MetricsHover) {
  const ts = host.points[spanMiddle(hoverSpan(host.points.length, hover))]?.ts;
  return ts === undefined ? "" : clock.value.format(new Date(ts));
}

const bucketLabel = computed(() => {
  const seconds = Math.round(bucket.value / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
});
</script>
