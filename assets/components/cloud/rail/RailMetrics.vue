<template>
  <!--
    The history behind the live chart.

    Dozzle keeps 300 samples in the browser and nothing behind them, so "was it
    like this an hour ago" has no local answer to gate. This reads back the
    samples this instance has been pushing all along.
  -->
  <!-- The gutter is reserved so a scrollbar coming and going (a loading line is
       shorter than the charts) never changes the width `columns` is read from:
       with classic scrollbars that refetched, which showed the loading line,
       which hid the scrollbar, and so on. -->
  <div ref="root" class="h-full [scrollbar-gutter:stable] overflow-y-auto p-4">
    <p v-if="!container" class="text-base-content/60 text-sm">{{ $t("cloud-rail.metrics-no-container") }}</p>

    <template v-else>
      <div class="mb-4 flex items-center gap-2">
        <span class="truncate font-mono text-sm font-semibold">{{ container.name }}</span>
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
      <p v-else-if="failed" class="text-base-content/60 text-sm">{{ $t("cloud-rail.metrics-failed") }}</p>
      <p v-else-if="!points.length" class="text-base-content/60 text-sm">{{ $t("cloud-rail.metrics-empty") }}</p>

      <template v-else>
        <!-- One pointer for the whole stack: hovering any chart marks the same
             moment on all of them, so "CPU rose when the network did" is read
             straight down the column. Leaving falls back to each chart's
             summary for the window. -->
        <section v-for="chart in charts" :key="chart.key" class="mb-5 last:mb-0">
          <div class="mb-1 flex items-baseline justify-between gap-2">
            <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ chart.label }}</span>
            <span class="truncate font-mono text-xs tabular-nums">
              <span class="font-semibold">{{ readout(chart).main }}</span>
              <span class="text-base-content/40">{{ readout(chart).aside }}</span>
            </span>
          </div>
          <BarChart
            :chart-data="chart.data"
            :bar-class="chart.colorClass"
            :hover-index="hovered?.index ?? null"
            class="h-16"
            @hover-value="(_: number, index: number, bars: number) => (hovered = { index, bars })"
            @hover-end="hovered = null"
          />
        </section>

        <MetricsTimeline
          class="mt-3"
          :from="range.from"
          :to="range.to"
          :alerts="alerts"
          :deploys="deploys"
          :format="(ms: number) => clock.format(new Date(ms))"
        />

        <!-- The hovered moment is named once, here, rather than in every header,
             with whatever happened inside that bar. -->
        <p class="text-base-content/40 mt-4 truncate font-mono text-xs tabular-nums">
          <template v-if="hovered">{{ timeOf(hovered) }} · </template>
          <template v-if="hovered && eventIn(hovered)"
            ><span class="text-base-content/70">{{ eventIn(hovered) }}</span> ·
          </template>
          {{ $t("cloud-rail.metrics-bucket", { duration: bucketLabel }) }}
        </p>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { BarDataPoint } from "@/components/ui/BarChart.vue";
import type { CloudAlert } from "@/composable/cloud/cloudAlerts";
import { mergeRecentDeploys, useRecentDeploys } from "@/composable/cloud/recentDeploys";
import { updateKind, updateLabels } from "@/models/ContainerUpdate";

// The max and rate fields are absent from an older cloud. A missing max falls back
// to the average; a missing rate hides its chart rather than drawing zeros that
// read as "no traffic".
type MetricPoint = {
  ts: number;
  cpu: number;
  memory: number;
  memoryUsage: number;
  cpuMax?: number;
  memoryUsageMax?: number;
  networkRx?: number;
  networkTx?: number;
  diskRead?: number;
  diskWrite?: number;
};

const { t } = useI18n();
const view = useViewContext();
const store = useContainerStore();
const { fetchAlerts } = useCloudAlerts();
const { cloudDeploys, refresh: refreshCloudDeploys } = useRecentDeploys();
const { metricsLookbackHours, ensureCloudStatus } = useCloudConfig();

// The plan decides whether the week is on offer, so the panel needs the status
// even on a page where nothing else has asked for it. Shared and cached.
onMounted(ensureCloudStatus);
// What Cloud remembers past a restart; Dozzle's own updates are already in the store.
onMounted(refreshCloudDeploys);

// One container at a time: a merged view is many histories, and stacking them
// in a 550px column reads as noise. The first in view is the one on screen.
const containerId = computed(() => view.value.containers[0]?.id);
const container = computed(() => (containerId.value ? store.allContainersById[containerId.value] : undefined));
// Cloud keys these by name, and the host is half that identity, so the lookup
// goes through the host route the rest of the container API uses.
const source = computed(() =>
  container.value ? `/api/cloud/hosts/${container.value.host}/containers/${container.value.id}/metrics` : "",
);

// A day is the question this panel is actually for: the live chart already
// covers the last few minutes, so opening on an hour showed a slower copy of
// what is on screen. A week is retention rather than a wider chart, so it is
// only offered where the plan keeps that much.
const windows = computed(() => [
  { value: "1h", label: t("cloud-rail.window-1h") },
  { value: "6h", label: t("cloud-rail.window-6h") },
  { value: "24h", label: t("cloud-rail.window-24h") },
  // Go parses hours, not days, so the week travels as 168h.
  ...(metricsLookbackHours.value >= 168 ? [{ value: "168h", label: t("cloud-rail.window-7d") }] : []),
]);
const window = ref("24h");

// A shorter lookback (or the status arriving late) must not leave the tabs pointing at
// a window that is no longer on offer.
watch(windows, (options) => {
  if (!options.some((o) => o.value === window.value)) window.value = "24h";
});

const points = ref<MetricPoint[]>([]);
const alerts = ref<CloudAlert[]>([]);
const bucket = ref(0);
const loading = ref(false);
const failed = ref(false);

// As many buckets as the chart has columns. Asking for a fixed 120 left a
// ~100-column chart folding pairs into 60 bars spread across the width, with a
// gap twice the bar between each; one bucket per column draws at the same
// density as every other stat chart. 5px is BarChart's bar plus gap; the width is
// the content box, so the p-4 is already out of it.
const root = ref<HTMLElement | null>(null);
const { width: rootWidth } = useElementSize(root);
const columns = ref(0);
const measure = () => (columns.value = Math.min(1000, Math.max(1, Math.floor(rootWidth.value / 5))));
// Settled before refetching, so dragging the rail wider does not fire a request
// per five pixels.
watchDebounced(rootWidth, (w) => w > 0 && measure(), { debounce: 300 });

// Only the newest request may write. Switching windows, or resizing the rail
// mid-load, leaves an older request in flight, and whichever answered last used
// to win: a 7d tab could end up showing the hour it was switched away from.
let latest = 0;

// The alerts for exactly the span the bars cover. Asked after the points, so
// the window is the one cloud answered for, and not awaited, so the charts never
// wait on it. Best effort: a failure is an empty timeline, never a failed panel.
async function loadAlerts(current: () => boolean) {
  const { from, to } = range.value;
  const id = container.value?.id;
  const found = id && to > from ? (await fetchAlerts([id], new Date(from), new Date(to))).alerts : [];
  if (current()) alerts.value = found;
}

async function load() {
  // Not measured yet: the first width to arrive sets `columns`, which loads.
  if (!source.value || columns.value === 0) return;
  const request = ++latest;
  const current = () => request === latest;
  loading.value = true;
  failed.value = false;
  alerts.value = [];
  try {
    const res = await fetch(withBase(`${source.value}?window=${window.value}&buckets=${columns.value}`));
    if (!current()) return;
    if (!res.ok) {
      // 503 is an unlinked instance, which the rail is not shown on anyway.
      failed.value = res.status !== 503;
      points.value = [];
      return;
    }
    const body = await res.json();
    if (!current()) return;
    points.value = body.points ?? [];
    bucket.value = body.bucket ?? 0;
    void loadAlerts(current);
  } catch {
    if (!current()) return;
    failed.value = true;
    points.value = [];
  } finally {
    if (current()) loading.value = false;
  }
}

watch([source, window, columns], load, { immediate: true });
// The panel mounts before it has a width. The first one is taken at once rather
// than after the debounce, so opening the panel does not wait on it.
watch(rootWidth, (w) => w > 0 && columns.value === 0 && measure(), { immediate: true });

// The bar under the pointer, shared by every chart: they draw the same window at
// the same width, so bar `index` is the same moment on all of them, and one
// pointer reads the whole column.
type Hover = { index: number; bars: number };
const hovered = ref<Hover | null>(null);

watch([source, window], () => (hovered.value = null));

type Readout = { main: string; aside: string };
type Chart = {
  key: string;
  label: string;
  colorClass: string;
  data: BarDataPoint[];
  // What the header says with the pointer elsewhere, and on a hovered bar.
  rest: Readout;
  at: (span: [number, number]) => Readout;
};

const pct = (v: number) => `${v.toFixed(1)}%`;
const bytes = (v: number) => formatBytes(v, { short: true, decimals: 1 });
const mean = (xs: number[], [from, to]: [number, number]) =>
  xs.slice(from, to).reduce((a, b) => a + b, 0) / Math.max(1, to - from);

// A level (CPU, memory) rests on its peak: the highest sample cloud saw, not the
// highest average, which shrank as the buckets widened and read a week lower than
// the day inside it. An older cloud sends no max, and the average stands in.
function levelChart(
  key: string,
  label: string,
  colorClass: string,
  format: (v: number) => string,
  avgOf: (p: MetricPoint) => number,
  maxOf: (p: MetricPoint) => number | undefined,
): Chart {
  const values = points.value.map(avgOf);
  const max = points.value.map(maxOf);
  const peaks = values.map((v, i) => max[i] ?? v);
  const hasMax = max.some((m) => m !== undefined);
  const peakWord = t("cloud-rail.metrics-peak");
  return {
    key,
    label,
    colorClass,
    data: values.map((v, i) =>
      hasMax ? { percent: v, value: v, peakPercent: peaks[i], peak: peaks[i] } : { percent: v, value: v },
    ),
    rest: { main: format(Math.max(0, ...peaks)), aside: ` / ${peakWord}` },
    at: (span) => ({
      main: format(mean(values, span)),
      aside: hasMax ? ` · ${peakWord} ${format(Math.max(...peaks.slice(...span)))}` : "",
    }),
  };
}

// I/O rests on what moved over the whole window: a peak rate says little about
// a counter, and the total is the number people ask for. A bar is both
// directions together; the readout splits them.
function ioChart(
  key: string,
  label: string,
  colorClass: string,
  inOf: (p: MetricPoint) => number | undefined,
  outOf: (p: MetricPoint) => number | undefined,
): Chart | undefined {
  // Absent on an older cloud: hidden, rather than a flat line that reads as idle.
  if (!points.value.some((p) => inOf(p) !== undefined || outOf(p) !== undefined)) return undefined;
  const ins = points.value.map((p) => inOf(p) ?? 0);
  const outs = points.value.map((p) => outOf(p) ?? 0);
  const total = (xs: number[]) => xs.reduce((a, b) => a + b, 0) * (bucket.value / 1000);
  return {
    key,
    label,
    colorClass,
    data: ins.map((v, i) => ({ percent: v + outs[i], value: v + outs[i] })),
    rest: { main: `↑${bytes(total(outs))} ↓${bytes(total(ins))}`, aside: ` / ${windowLabel.value}` },
    at: (span) => ({ main: `↑${formatRate(mean(outs, span))} ↓${formatRate(mean(ins, span))}`, aside: "" }),
  };
}

const charts = computed(() =>
  [
    levelChart(
      "cpu",
      t("label.cpu"),
      "text-primary",
      pct,
      (p) => p.cpu,
      (p) => p.cpuMax,
    ),
    // Bytes rather than percent: it is what people ask about, and a limit-less
    // container has no percent worth reading.
    levelChart(
      "memory",
      t("label.mem"),
      "text-secondary",
      bytes,
      (p) => p.memoryUsage,
      (p) => p.memoryUsageMax,
    ),
    ioChart(
      "network",
      t("label.network"),
      "text-info",
      (p) => p.networkRx,
      (p) => p.networkTx,
    ),
    ioChart(
      "disk",
      t("label.disk-io"),
      "text-base-content/60",
      (p) => p.diskRead,
      (p) => p.diskWrite,
    ),
  ].filter((c): c is Chart => c !== undefined),
);

const readout = (chart: Chart) => (hovered.value ? chart.at(spanOf(hovered.value)) : chart.rest);

// The points behind a hovered bar. BarChart folds the series into equal runs of
// ceil(points / bars), so this is the same run it averaged to draw the bar.
function spanOf(hover: Hover): [number, number] {
  const size = Math.ceil(points.value.length / Math.max(1, hover.bars));
  const from = hover.index * size;
  return [from, Math.min(points.value.length, from + size)];
}

const windowLabel = computed(() => windows.value.find((o) => o.value === window.value)?.label ?? window.value);

// Follows the reader's own clock preference, like every other time in the app.
const hourCycle = computed(() => (hourStyle.value === "12" ? "h12" : hourStyle.value === "24" ? "h23" : undefined));
// Past a day the clock alone is ambiguous: "09:14" on a week of bars could be
// any of seven mornings, so the date comes along.
const clock = computed(
  () =>
    new Intl.DateTimeFormat(undefined, {
      month: window.value === "1h" || window.value === "6h" ? undefined : "short",
      day: window.value === "1h" || window.value === "6h" ? undefined : "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: hourCycle.value,
    }),
);

// The time the bars cover: from the first bucket's start to the last one's end.
const range = computed(() => {
  const first = points.value[0];
  const last = points.value.at(-1);
  return first && last ? { from: first.ts, to: last.ts + bucket.value } : { from: 0, to: 0 };
});

// This container's deploys: Dozzle's own and the ones only Cloud remembers.
// Matched by host and name, because a deploy replaces the id.
const deploys = computed(() => {
  const c = container.value;
  if (!c) return [];
  return mergeRecentDeploys(store.recentUpdates, cloudDeploys.value).filter(
    (d) => d.host === c.host && d.name === c.name,
  );
});

// What happened inside a hovered bar, for the footer. The newest one when
// several share a bar: it is the one the bar's level most likely reflects.
function eventIn(hover: Hover) {
  const [i, j] = spanOf(hover);
  const start = points.value[i]?.ts;
  const end = points.value[j - 1]?.ts;
  if (start === undefined || end === undefined) return "";
  const inside = (ts: number) => ts >= start && ts < end + bucket.value;
  const deploy = deploys.value.filter((d) => inside(Date.parse(d.at))).at(0);
  const alert = alerts.value.filter((a) => inside(a.ts / 1e6)).at(-1);
  if (alert && (!deploy || alert.ts / 1e6 > Date.parse(deploy.at))) return alert.headline;
  return deploy ? `${t(`update-marker.${updateKind(deploy)}`)} ${updateLabels(deploy).to}` : "";
}

// A bar covers a run of points; its middle is the honest moment to name.
function timeOf(hover: Hover) {
  const [from, to] = spanOf(hover);
  const point = points.value[Math.min(points.value.length - 1, Math.floor((from + to - 1) / 2))];
  return point ? clock.value.format(new Date(point.ts)) : "";
}

// Said out loud so a flat line reads as an average rather than a missing sample.
const bucketLabel = computed(() => {
  const seconds = Math.round(bucket.value / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
});
</script>
