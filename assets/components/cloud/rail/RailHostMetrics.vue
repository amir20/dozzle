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
      <section v-for="host in hostSeries" :key="host.id" class="mb-6 last:mb-0" :data-host="host.id">
        <h3 class="mb-2 truncate font-mono text-sm font-semibold">{{ host.name }}</h3>
        <div v-for="chart in host.charts" :key="chart.key" class="mb-3 last:mb-0">
          <div class="mb-1 flex items-baseline justify-between gap-2">
            <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ chart.label }}</span>
            <span class="truncate font-mono text-xs tabular-nums">
              <span class="font-semibold">{{ chart.peak }}</span>
              <span class="text-base-content/40"> / {{ $t("cloud-rail.metrics-peak") }}</span>
            </span>
          </div>
          <p class="text-base-content/40 mb-1 truncate font-mono text-xs tabular-nums">
            <span class="text-base-content/60 font-semibold">{{ chart.summary }}</span>{{
              chart.limit ? ` / ${chart.limit}` : ""
            }}
          </p>
          <BarChart :chart-data="chart.data" :bar-class="chart.colorClass" :max="chart.ceiling" class="h-12" />
        </div>
      </section>

      <p class="text-base-content/40 mt-4 truncate font-mono text-xs tabular-nums">
        {{ $t("cloud-rail.metrics-bucket", { duration: bucketLabel }) }}
      </p>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { BarDataPoint } from "@/components/ui/BarChart.vue";
import type { Container } from "@/models/Container";
import { hostLiveSeries, MIN_CLOUD_POINTS } from "@/composable/cloud/liveMetrics";

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
const avg = (xs: number[]) => xs.reduce((a, b) => a + b, 0) / Math.max(1, xs.length);

type Chart = {
  key: string;
  label: string;
  colorClass: string;
  data: BarDataPoint[];
  ceiling?: number;
  peak: string;
  summary: string;
  limit?: string;
};

// A host Cloud remembers but Dozzle no longer connects to keeps its id as its
// name: its history is still real, and the id is what it was known by.
const hostSeries = computed(() =>
  series.value.map(({ hostId, points }) => {
    const host = hosts.value[hostId];
    const cpu = points.map((p) => p.cpu);
    const memory = points.map((p) => p.memoryUsage);
    const memTotal = host?.memTotal ?? 0;
    const charts: Chart[] = [
      // The sum of each container's CPU, which reads against the host's cores
      // for a container with no limit. Scaled to its own peak: a container with
      // a limit reports against that, so the sum can pass 100.
      {
        key: "cpu",
        label: t("label.cpu"),
        colorClass: "text-primary",
        data: cpu.map((v) => ({ percent: v, value: v })),
        peak: pct(Math.max(0, ...cpu)),
        summary: `${t("analytics.avg")} ${pct(avg(cpu))}`,
      },
      // Drawn against the host's memory when Dozzle knows it, like the host card.
      {
        key: "memory",
        label: t("label.mem"),
        colorClass: "text-secondary",
        data: memory.map((v) => ({ percent: memTotal ? (v / memTotal) * 100 : v, value: v })),
        ceiling: memTotal ? 100 : undefined,
        peak: bytes(Math.max(0, ...memory)),
        summary: `${t("analytics.avg")} ${bytes(avg(memory))}`,
        limit: memTotal ? bytes(memTotal) : undefined,
      },
    ];
    return { id: hostId, name: host?.name ?? hostId, charts };
  }),
);

const bucketLabel = computed(() => {
  const seconds = Math.round(bucket.value / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
});
</script>
