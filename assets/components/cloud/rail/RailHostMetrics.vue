<template>
  <!--
    The Metrics tab about hosts rather than one container: every host this
    instance pushes to Cloud, summed, and then one host at a time. Everything
    here is Cloud's history; the live numbers are already on the home page.
  -->
  <div ref="root" class="h-full [scrollbar-gutter:stable] overflow-y-auto p-4">
    <!-- Which hosts, and over what window. A few hosts get one segment each;
         past four the same choice is a menu with a filter, so a fleet of twenty
         is never a wall of buttons. -->
    <div class="mb-4 flex items-center gap-2">
      <div
        v-if="hosts.length > 1 && hosts.length <= SEGMENT_LIMIT"
        role="group"
        class="border-base-content/15 bg-base-200/40 inline-flex max-w-full min-w-0 rounded-full border p-0.5 text-xs"
        :aria-label="$t('cloud-rail.scope')"
        data-host-segments
      >
        <button
          v-for="segment in segments"
          :key="segment.id ?? ''"
          type="button"
          class="min-w-0 truncate rounded-full px-3 py-1 transition-colors"
          :class="[
            selected === segment.id ? 'bg-base-300 font-semibold' : 'text-base-content/60 hover:bg-base-300',
            { 'font-mono': segment.id },
          ]"
          :aria-pressed="selected === segment.id"
          @click="selected = segment.id"
        >
          {{ segment.name }}
        </button>
      </div>

      <Popover
        v-else-if="hosts.length > SEGMENT_LIMIT"
        placement="bottom-start"
        panel-class="rounded-box border-base-content/10 bg-base-200 w-72 border p-2 shadow-lg"
        data-host-menu
        @close="filter = ''"
      >
        <template #trigger>
          <button type="button" class="btn btn-sm gap-2 font-normal">
            <mdi:server class="size-4 opacity-60" />
            <span :class="{ 'font-mono': current }">{{
              current ? current.name : $t("cloud-rail.scope-all-hosts")
            }}</span>
            <span v-if="!current" class="text-base-content/40 font-mono">{{ hosts.length }}</span>
            <mdi:chevron-down class="size-4 opacity-60" />
          </button>
        </template>
        <template #default="{ close }">
          <label class="input input-sm mb-1.5 w-full">
            <mdi:magnify class="size-4 opacity-60" />
            <input
              v-model="filter"
              type="search"
              :placeholder="$t('cloud-rail.metrics-filter-hosts', { n: hosts.length })"
            />
          </label>
          <div class="max-h-80 overflow-y-auto">
            <template v-if="!filter">
              <button
                type="button"
                class="hover:bg-base-300 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors"
                :class="{ 'bg-base-300': !selected }"
                @click="pick(null, close)"
              >
                <mdi:server-network class="size-4 opacity-60" />
                <span class="flex-1">{{ $t("cloud-rail.scope-all-hosts") }}</span>
              </button>
              <div class="bg-base-content/10 my-1 h-px"></div>
            </template>
            <!-- Busiest first, so the host worth opening is at the top of a long list. -->
            <button
              v-for="host in menuHosts"
              :key="host.id"
              type="button"
              class="hover:bg-base-300 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors"
              :class="{ 'bg-base-300': selected === host.id }"
              data-host-option
              @click="pick(host.id, close)"
            >
              <span class="min-w-0 flex-1 truncate font-mono text-sm">{{ host.name }}</span>
              <span v-if="host.hot" class="bg-warning size-1.5 shrink-0 rounded-full"></span>
              <span class="text-base-content/60 w-12 shrink-0 text-right font-mono text-xs tabular-nums">
                {{ pct(host.stats.cpu.avg) }}
              </span>
            </button>
            <p v-if="!menuHosts.length" class="text-base-content/60 px-2 py-1.5 text-sm">
              {{ $t("cloud-rail.metrics-no-host-match") }}
            </p>
          </div>
        </template>
      </Popover>

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

    <p v-if="loading && !hosts.length" class="text-base-content/60 text-sm">{{ $t("cloud-rail.metrics-loading") }}</p>
    <p v-else-if="failed" class="text-base-content/60 text-sm">{{ $t("cloud-rail.metrics-failed") }}</p>
    <p v-else-if="!view" class="text-base-content/60 text-sm">{{ $t("cloud-rail.metrics-hosts-empty") }}</p>

    <template v-else>
      <!-- What the numbers below are a total of, said once, so nobody reads a
           host's sum as one container's. -->
      <div class="mb-4 flex items-center gap-3" data-subject>
        <div class="bg-base-content/5 shrink-0 rounded-full p-2">
          <mdi:server v-if="current" class="size-5 opacity-70" />
          <mdi:server-network v-else class="size-5 opacity-70" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <template v-if="current">
              <span class="truncate font-mono text-base font-semibold">{{ current.name }}</span>
              <span v-if="current.type" class="status-pill" :class="current.type === 'agent' ? 'secondary' : 'neutral'">
                {{ current.type }}
              </span>
            </template>
            <span v-else class="text-base font-semibold">{{ $t("cloud-rail.scope-all-hosts") }}</span>
          </div>
          <p class="text-base-content/60 text-xs">
            <template v-if="current">
              {{ $t("cloud-rail.metrics-whole-host") }} ·
              {{ $t("cloud-rail.metrics-containers-summed", { n: current.containers }) }}
            </template>
            <template v-else>
              {{ $t("label.host-count", hosts.length) }} ·
              {{ $t("cloud-rail.metrics-containers-summed", { n: totalContainers }) }}
            </template>
          </p>
        </div>
        <button
          v-if="current && hosts.length > 1"
          type="button"
          class="btn btn-ghost btn-xs text-base-content/60"
          @click="selected = null"
        >
          <mdi:arrow-left class="size-3.5" /> {{ $t("cloud-rail.scope-all-hosts") }}
        </button>
      </div>

      <!-- One pointer for the whole stack: hovering any chart marks the same
           moment on all of them, as in the container history. -->
      <section v-for="chart in charts" :key="chart.key" class="mb-5 last:mb-0" :data-chart="chart.key">
        <div class="mb-1 flex items-baseline justify-between gap-2">
          <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ chart.label }}</span>
          <span class="truncate font-mono text-xs tabular-nums">
            <span class="font-semibold">{{ readout(chart).main }}</span>
            <span class="text-base-content/40">{{ readout(chart).aside }}</span>
          </span>
        </div>
        <p class="text-base-content/40 mb-1 truncate font-mono text-xs tabular-nums" data-summary>
          <span class="text-base-content/60 font-semibold">{{ chart.summary }}</span>
        </p>
        <BarChart
          :chart-data="chart.data"
          :bar-class="chart.colorClass"
          :hover-index="hovered?.index ?? null"
          class="h-16"
          @hover-value="(_: number, index: number, bars: number) => (hovered = { index, bars })"
          @hover-end="hovered = null"
        />
      </section>

      <div class="text-base-content/40 mt-3 flex justify-between font-mono text-xs tabular-nums" data-range>
        <span>{{ clock.format(new Date(view.range.from)) }}</span>
        <span>{{ clock.format(new Date(view.range.to)) }}</span>
      </div>
      <p class="text-base-content/40 mt-2 truncate font-mono text-xs tabular-nums" data-moment>
        <template v-if="hovered">{{ timeOf(hovered) }} · </template>
        {{ $t("cloud-rail.metrics-bucket", { duration: bucketLabel }) }}
      </p>

      <!-- Every host: which one is doing it, ranked by the chosen metric. A row
           narrows the whole panel to that host. -->
      <template v-if="!current && hosts.length > 1">
        <div class="mt-6 mb-2 flex items-center gap-2">
          <h4 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
            {{ $t("cloud-rail.metrics-hosts") }}
          </h4>
          <MetricTabs v-model="hostRank" :metrics="metrics" class="ml-auto" />
        </div>
        <div class="border-base-content/15 bg-base-200/40 rounded-lg border p-1" data-host-ranking>
          <button
            v-for="row in rankedHosts"
            :key="row.id"
            type="button"
            class="hover:bg-base-300 flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left transition-colors"
            @click="selected = row.id"
          >
            <span class="min-w-0 flex-1 truncate font-mono text-sm">{{ row.name }}</span>
            <span
              v-if="row.hot"
              class="bg-warning size-1.5 shrink-0 rounded-full"
              :title="$t('cloud-rail.metrics-hot')"
            ></span>
            <div class="bg-base-content/10 h-1.5 w-16 shrink-0 overflow-hidden rounded-full">
              <div
                class="h-full rounded-full"
                :class="
                  hostRank === 'cpu' && row.share > 90
                    ? 'bg-error'
                    : hostRank === 'cpu' && row.share > 70
                      ? 'bg-warning'
                      : 'bg-primary'
                "
                :style="{ width: `${row.share}%` }"
              ></div>
            </div>
            <span class="shrink-0 text-right font-mono text-xs whitespace-nowrap tabular-nums">
              <span class="font-semibold">{{ row.avg }}</span><span class="text-base-content/40">
                · {{ row.peak }}</span
              >
            </span>
            <mdi:chevron-right class="size-4 shrink-0 opacity-40" />
          </button>
        </div>
        <button
          v-if="hosts.length > HOST_ROWS"
          type="button"
          class="text-base-content/60 hover:text-base-content mt-1.5 text-xs transition-colors"
          @click="showAllHosts = !showAllHosts"
        >
          {{
            showAllHosts
              ? $t("cloud-rail.metrics-show-fewer")
              : $t("cloud-rail.metrics-show-all-hosts", { n: hosts.length })
          }}
        </button>
      </template>

      <!-- What is using it: the heaviest containers over the window, each one
           click from its page. Absent for a reader with a container filter, who
           gets the totals but not the names. -->
      <template v-if="topRows">
        <div class="mt-6 mb-2 flex items-center gap-2">
          <h4 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
            {{ $t("cloud-rail.metrics-top") }}
          </h4>
          <MetricTabs v-model="topRank" :metrics="metrics" class="ml-auto" />
        </div>
        <div class="border-base-content/15 bg-base-200/40 rounded-lg border p-1" data-top>
          <component
            :is="row.to ? RouterLink : 'div'"
            v-for="row in topRows"
            :key="`${row.hostId}/${row.name}`"
            :to="row.to"
            class="flex items-center gap-3 rounded-md px-2 py-1.5"
            :class="{ 'hover:bg-base-300 transition-colors': row.to }"
            @click="row.to && closeRail()"
          >
            <span class="min-w-0 flex-1 truncate font-mono text-sm">
              {{ row.name
              }}<span v-if="!current && hosts.length > 1" class="text-base-content/40"> · {{ row.hostName }}</span>
            </span>
            <div class="bg-base-content/10 h-1.5 w-16 shrink-0 overflow-hidden rounded-full">
              <div class="bg-primary h-full rounded-full" :style="{ width: `${row.share}%` }"></div>
            </div>
            <span class="shrink-0 text-right font-mono text-xs whitespace-nowrap tabular-nums">
              <span class="font-semibold">{{ row.avg }}</span><span class="text-base-content/40">
                · {{ row.peak }}</span
              >
            </span>
          </component>
          <p v-if="!topRows.length" class="text-base-content/60 px-2 py-1.5 text-sm">
            {{ $t("cloud-rail.metrics-top-empty") }}
          </p>
        </div>
        <p class="text-base-content/40 mt-2 text-xs">
          {{ $t("cloud-rail.metrics-top-note", { window: windowLabel }) }}
        </p>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { RouterLink } from "vue-router";
import type { BarDataPoint } from "@/components/ui/BarChart.vue";
import { hoverSpan, spanMiddle, useMetricsClock, windowClock, type MetricsHover } from "@/composable/cloud/metricsTime";
import {
  aggregateHosts,
  hostStats,
  mergeTop,
  type HostPoint,
  type HostSeries,
  type Metric,
} from "@/composable/cloud/hostMetrics";

// Past this many hosts the picker is a menu rather than a row of segments.
const SEGMENT_LIMIT = 4;
// Hosts listed in the ranking before "show all".
const HOST_ROWS = 6;

const { t } = useI18n();
const { hosts: knownHosts } = useHosts();
const store = useContainerStore();
const viewContext = useViewContext();
const { closeRail } = useCloudRail();
const { metricsLookbackHours, ensureCloudStatus } = useCloudConfig();
onMounted(ensureCloudStatus);

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
const windowLabel = computed(() => windows.value.find((o) => o.value === window.value)?.label ?? window.value);

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

const pct = (v: number) => `${v.toFixed(1)}%`;
const bytes = (v: number) => formatBytes(v, { short: true, decimals: 1 });
const rate = (v: number) => formatRate(v);
const format: Record<Metric, (v: number) => string> = { cpu: pct, memory: bytes, network: rate, disk: rate };

// Every host Cloud has history for. One Dozzle no longer connects to keeps its
// id as its name: its history is still real, and the id is what it was known by.
const hosts = computed(() =>
  series.value
    .filter((s) => s.points.length > 0)
    .map((s) => {
      const known = knownHosts.value[s.hostId];
      const stats = hostStats(s.points);
      return {
        id: s.hostId,
        name: known?.name ?? s.hostId,
        type: known?.type,
        containers: s.containers ?? 0,
        points: s.points,
        top: s.top,
        stats,
        hot: stats.cpu.peak > 90,
      };
    }),
);
type Host = (typeof hosts.value)[number];
const totalContainers = computed(() => hosts.value.reduce((sum, h) => sum + h.containers, 0));

// The host being read, or null for all of them. On a container page it opens on
// that container's host, the one the reader just widened from.
const selected = ref<string | null>(null);
const pageHost = computed(() => {
  const id = viewContext.value.containers[0]?.id;
  return id ? store.allContainersById[id]?.host : undefined;
});
// Seeded once, when the first answer arrives, and again only when the page
// changes: a window switch that changes which hosts answered must not undo a
// reader who picked "All hosts".
let seeded = false;
const seed = (host: string | undefined) => {
  if (host && hosts.value.some((h) => h.id === host)) selected.value = host;
};
watch(
  () => hosts.value.length > 0,
  (ready) => {
    if (!ready || seeded) return;
    seeded = true;
    seed(pageHost.value);
  },
  { immediate: true },
);
watch(pageHost, seed);
// A single host has nothing to sum across, and a host that left the answer is not selectable.
const current = computed<Host | undefined>(() =>
  hosts.value.length === 1 ? hosts.value[0] : hosts.value.find((h) => h.id === selected.value),
);

const segments = computed(() => [
  { id: null as string | null, name: t("cloud-rail.scope-all-hosts") },
  ...hosts.value.map((h) => ({ id: h.id as string | null, name: h.name })),
]);

const filter = ref("");
const menuHosts = computed(() =>
  [...hosts.value]
    .sort((a, b) => b.stats.cpu.avg - a.stats.cpu.avg)
    .filter((h) => h.name.toLowerCase().includes(filter.value.trim().toLowerCase())),
);
function pick(id: string | null, close: () => void) {
  selected.value = id;
  filter.value = "";
  close();
}

// The series on screen: one host's, or every host's combined. Across hosts CPU is
// the average host's load, because adding percentages of different core counts
// says nothing; the rest are plain sums.
const view = computed(() => {
  const points: HostPoint[] = current.value ? current.value.points : aggregateHosts(hosts.value.map((h) => h.points));
  if (points.length < 2) return undefined;
  const first = points[0];
  const last = points.at(-1)!;
  return { points, range: { from: first.ts, to: last.ts + bucket.value } };
});

const hovered = ref<MetricsHover | null>(null);
watch([window, selected], () => (hovered.value = null));

type Readout = { main: string; aside: string };
type Chart = {
  key: string;
  label: string;
  colorClass: string;
  data: BarDataPoint[];
  summary: string;
  rest: Readout;
  at: (span: [number, number]) => Readout;
};

const mean = (xs: number[], [from, to]: [number, number]) =>
  xs.slice(from, to).reduce((a, b) => a + b, 0) / Math.max(1, to - from);

// A level rests on its peak: the highest Cloud saw, not the highest average. An
// older cloud sends no max, and the average stands in.
function levelChart(
  key: string,
  label: string,
  colorClass: string,
  fmt: (v: number) => string,
  avgOf: (p: HostPoint) => number,
  maxOf: (p: HostPoint) => number | undefined,
): Chart {
  const points = view.value!.points;
  const values = points.map(avgOf);
  const max = points.map(maxOf);
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
    summary: `${t("analytics.avg")} ${fmt(mean(values, [0, values.length]))}`,
    rest: { main: fmt(Math.max(0, ...peaks)), aside: ` / ${peakWord}` },
    at: (span) => ({
      main: fmt(mean(values, span)),
      aside: hasMax ? ` · ${peakWord} ${fmt(Math.max(...peaks.slice(...span)))}` : "",
    }),
  };
}

// I/O rests on what moved over the whole window; a bar is both directions, the
// readout splits them. Hidden when an older cloud sends no rates.
function ioChart(
  key: string,
  label: string,
  colorClass: string,
  inOf: (p: HostPoint) => number | undefined,
  outOf: (p: HostPoint) => number | undefined,
  marks: [string, string],
): Chart | undefined {
  const points = view.value!.points;
  if (!points.some((p) => inOf(p) !== undefined || outOf(p) !== undefined)) return undefined;
  const ins = points.map((p) => inOf(p) ?? 0);
  const outs = points.map((p) => outOf(p) ?? 0);
  const total = (xs: number[]) => xs.reduce((a, b) => a + b, 0) * (bucket.value / 1000);
  const [outMark, inMark] = marks;
  return {
    key,
    label,
    colorClass,
    data: ins.map((v, i) => ({ percent: v + outs[i], value: v + outs[i] })),
    summary: `${t("analytics.avg")} ${outMark}${rate(mean(outs, [0, outs.length]))} ${inMark}${rate(mean(ins, [0, ins.length]))}`,
    rest: { main: `${outMark}${bytes(total(outs))} ${inMark}${bytes(total(ins))}`, aside: ` / ${windowLabel.value}` },
    at: (span) => ({ main: `${outMark}${rate(mean(outs, span))} ${inMark}${rate(mean(ins, span))}`, aside: "" }),
  };
}

const charts = computed<Chart[]>(() => {
  if (!view.value) return [];
  return [
    levelChart(
      "cpu",
      current.value ? t("label.cpu") : t("cloud-rail.metrics-cpu-avg-host"),
      "text-primary",
      pct,
      (p) => p.cpu,
      (p) => p.cpuMax,
    ),
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
      ["↑", "↓"],
    ),
    ioChart(
      "disk",
      t("label.disk-io"),
      "text-base-content/60",
      (p) => p.diskRead,
      (p) => p.diskWrite,
      ["W ", "R "],
    ),
  ].filter((c): c is Chart => c !== undefined);
});

const readout = (chart: Chart) =>
  hovered.value && view.value ? chart.at(hoverSpan(view.value.points.length, hovered.value)) : chart.rest;

const clock = useMetricsClock(() => windowClock(window.value));
function timeOf(hover: MetricsHover) {
  const points = view.value?.points ?? [];
  const ts = points[spanMiddle(hoverSpan(points.length, hover))]?.ts;
  return ts === undefined ? "" : clock.value.format(new Date(ts));
}

const bucketLabel = computed(() => {
  const seconds = Math.round(bucket.value / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
});

// The ranking: each host's average and peak for one metric over the window. CPU
// is already a share of the host, so its bar is that share; the rest are drawn
// against the busiest host.
// An older cloud sends no rates: its Network and Disk lists would read as zero
// traffic, the flat line the rates were made optional to avoid, so they go.
const metrics = computed<Metric[]>(() => {
  const all = hosts.value.flatMap((h) => h.points);
  const has = (of: (p: HostPoint) => number | undefined) => all.some((p) => of(p) !== undefined);
  return [
    "cpu",
    "memory",
    ...(has((p) => p.networkRx ?? p.networkTx) ? (["network"] as const) : []),
    ...(has((p) => p.diskRead ?? p.diskWrite) ? (["disk"] as const) : []),
  ];
});
watch(metrics, (available) => {
  if (!available.includes(hostRank.value)) hostRank.value = "cpu";
  if (!available.includes(topRank.value)) topRank.value = "cpu";
});

const hostRank = ref<Metric>("cpu");
const showAllHosts = ref(false);
const rankedHosts = computed(() => {
  const metric = hostRank.value;
  const top = Math.max(1, ...hosts.value.map((h) => h.stats[metric].avg));
  const rows = hosts.value
    .map((h) => ({
      id: h.id,
      name: h.name,
      hot: h.hot,
      value: h.stats[metric].avg,
      share: metric === "cpu" ? Math.min(100, h.stats.cpu.avg) : (h.stats[metric].avg / top) * 100,
      avg: format[metric](h.stats[metric].avg),
      peak: format[metric](h.stats[metric].peak),
    }))
    .sort((a, b) => b.value - a.value);
  return showAllHosts.value ? rows : rows.slice(0, HOST_ROWS);
});

// The heaviest containers, for the host on screen or across all of them. A row
// links to its container when Dozzle still has one by that name on that host.
const topRank = ref<Metric>("cpu");
const topRows = computed(() => {
  const scope = current.value ? [current.value] : hosts.value;
  if (scope.some((h) => !h.top)) return undefined;
  const metric = topRank.value;
  const merged = mergeTop(
    scope.map((h) => ({ hostId: h.id, list: h.top![metric] })),
    5,
  );
  const first = Math.max(1e-9, merged[0]?.avg ?? 0);
  return merged.map((row) => {
    const container = store.containers.find((c) => c.host === row.hostId && c.name === row.name);
    return {
      ...row,
      hostName: hosts.value.find((h) => h.id === row.hostId)?.name ?? row.hostId,
      share: (row.avg / first) * 100,
      avg: format[metric](row.avg),
      peak: format[metric](row.peak),
      to: container ? { name: "/container/[id]" as const, params: { id: container.id } } : undefined,
    };
  });
});
</script>
