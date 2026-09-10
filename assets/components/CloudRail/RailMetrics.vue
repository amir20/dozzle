<template>
  <!--
    The history behind the live chart.

    Dozzle keeps 300 samples in the browser and nothing behind them, so "was it
    like this an hour ago" has no local answer to gate. This reads back the
    samples this instance has been pushing all along.
  -->
  <div class="h-full overflow-y-auto p-4">
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
        <!-- Hovering reads the chart out: a bar is a shape until it says what
             it was and when. Leaving falls back to the peak, which is the one
             number worth carrying while the pointer is elsewhere. -->
        <section
          v-for="chart in charts"
          :key="chart.key"
          class="mb-5 last:mb-0"
          @mouseleave="hovered[chart.key] = undefined"
        >
          <div class="mb-1 flex items-baseline justify-between gap-2">
            <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ chart.label }}</span>
            <span v-if="hovered[chart.key]" class="truncate font-mono text-xs tabular-nums">
              <span class="font-semibold">{{ chart.format(hovered[chart.key]!.value) }}</span>
              <span class="text-base-content/40"> · {{ timeOf(hovered[chart.key]!) }}</span>
            </span>
            <span v-else class="font-mono text-xs tabular-nums">
              <span class="font-semibold">{{ chart.format(chart.peak) }}</span>
              <span class="text-base-content/40"> / {{ $t("cloud-rail.metrics-peak") }}</span>
            </span>
          </div>
          <BarChart
            :chart-data="chart.data"
            :bar-class="`${chart.barClass} opacity-70 hover:opacity-100`"
            class="h-16"
            @hover-value="(value: number, index: number, bars: number) => (hovered[chart.key] = { value, index, bars })"
          />
        </section>

        <p class="text-base-content/40 mt-4 text-xs">
          {{ $t("cloud-rail.metrics-bucket", { duration: bucketLabel }) }}
        </p>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
type MetricPoint = { ts: number; cpu: number; memory: number; memoryUsage: number };

const { t } = useI18n();
const view = useViewContext();
const store = useContainerStore();
const { isPro, ensureCloudStatus } = useCloudConfig();

// The plan decides whether the week is on offer, so the panel needs the status
// even on a page where nothing else has asked for it. Shared and cached.
onMounted(ensureCloudStatus);

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
  ...(isPro.value ? [{ value: "168h", label: t("cloud-rail.window-7d") }] : []),
]);
const window = ref("24h");

// Losing pro (or the status arriving late) must not leave the tabs pointing at
// a window that is no longer on offer.
watch(windows, (options) => {
  if (!options.some((o) => o.value === window.value)) window.value = "24h";
});

const points = ref<MetricPoint[]>([]);
const bucket = ref(0);
const loading = ref(false);
const failed = ref(false);

async function load() {
  if (!source.value) return;
  loading.value = true;
  failed.value = false;
  try {
    const res = await fetch(withBase(`${source.value}?window=${window.value}`));
    if (!res.ok) {
      // 503 is an unlinked instance, which the rail is not shown on anyway.
      failed.value = res.status !== 503;
      points.value = [];
      return;
    }
    const body = await res.json();
    points.value = body.points ?? [];
    bucket.value = body.bucket ?? 0;
  } catch {
    failed.value = true;
    points.value = [];
  } finally {
    loading.value = false;
  }
}

watch([source, window], load, { immediate: true });

// What the pointer is on, per chart. Keyed rather than one shared value so two
// charts never claim the same bar.
type Hover = { value: number; index: number; bars: number };
const hovered = ref<Record<string, Hover | undefined>>({});

watch([source, window], () => (hovered.value = {}));

const charts = computed(() => [
  {
    key: "cpu",
    label: t("label.cpu"),
    barClass: "bg-primary",
    peak: Math.max(0, ...points.value.map((p) => p.cpu)),
    format: (v: number) => `${v.toFixed(1)}%`,
    data: points.value.map((p) => ({ percent: p.cpu, value: p.cpu })),
  },
  {
    key: "memory",
    label: t("label.mem"),
    barClass: "bg-secondary",
    // Bytes rather than percent: it is what the hover reads out, and a peak in
    // a different unit than the value it qualifies reads as a bug.
    peak: Math.max(0, ...points.value.map((p) => p.memoryUsage)),
    format: (v: number) => formatBytes(v, { short: true, decimals: 1 }),
    data: points.value.map((p) => ({ percent: p.memory, value: p.memoryUsage })),
  },
]);

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

// The bars are buckets of the series, so the hovered one covers a span of
// points. Its middle is the honest moment to name.
function timeOf(hover: Hover) {
  if (!points.value.length || hover.bars === 0) return "";
  const per = points.value.length / hover.bars;
  const index = Math.min(points.value.length - 1, Math.floor((hover.index + 0.5) * per));
  return clock.value.format(new Date(points.value[index].ts));
}

// Said out loud so a flat line reads as an average rather than a missing sample.
const bucketLabel = computed(() => {
  const seconds = Math.round(bucket.value / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
});
</script>
