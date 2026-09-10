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
        <section v-for="chart in charts" :key="chart.key" class="mb-5 last:mb-0">
          <div class="mb-1 flex items-baseline justify-between">
            <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ chart.label }}</span>
            <span class="font-mono text-xs">
              <span class="font-semibold">{{ chart.peak }}</span>
              <span class="text-base-content/40"> / {{ $t("cloud-rail.metrics-peak") }}</span>
            </span>
          </div>
          <BarChart :chart-data="chart.data" :bar-class="chart.barClass" class="h-16" />
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

// One container at a time: a merged view is many histories, and stacking them
// in a 550px column reads as noise. The first in view is the one on screen.
const containerId = computed(() => view.value.containers[0]?.id);
const container = computed(() => (containerId.value ? store.allContainersById[containerId.value] : undefined));
// Cloud keys these by name, and the host is half that identity, so the lookup
// goes through the host route the rest of the container API uses.
const source = computed(() =>
  container.value ? `/api/cloud/hosts/${container.value.host}/containers/${container.value.id}/metrics` : "",
);

const windows = computed(() => [
  { value: "1h", label: t("cloud-rail.window-1h") },
  { value: "6h", label: t("cloud-rail.window-6h") },
  { value: "24h", label: t("cloud-rail.window-24h") },
]);
const window = ref("1h");

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

const charts = computed(() => {
  const cpuPeak = Math.max(0, ...points.value.map((p) => p.cpu));
  const memPeak = Math.max(0, ...points.value.map((p) => p.memory));

  return [
    {
      key: "cpu",
      label: t("label.cpu"),
      barClass: "bg-primary",
      peak: `${cpuPeak.toFixed(1)}%`,
      data: points.value.map((p) => ({ percent: p.cpu, value: p.cpu })),
    },
    {
      key: "memory",
      label: t("label.mem"),
      barClass: "bg-secondary",
      peak: `${memPeak.toFixed(1)}%`,
      data: points.value.map((p) => ({ percent: p.memory, value: p.memoryUsage })),
    },
  ];
});

// Said out loud so a flat line reads as an average rather than a missing sample.
const bucketLabel = computed(() => {
  const seconds = Math.round(bucket.value / 1000);
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
});
</script>
