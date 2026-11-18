<template>
  <!--
    What the metrics panel shows while Cloud has too little history to draw:
    the live window Dozzle already holds, in the same charts, with a line that
    says where it came from so it is never mistaken for Cloud's.
  -->
  <div data-live>
    <section v-for="item in charts" :key="item.id" class="mb-6 last:mb-0" :data-series="item.id">
      <h3 v-if="named" class="mb-2 truncate font-mono text-sm font-semibold">{{ item.name }}</h3>
      <div v-for="chart in item.charts" :key="chart.key" class="mb-3 last:mb-0">
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
        <BarChart
          :chart-data="chart.data"
          :bar-class="chart.colorClass"
          :max="chart.ceiling"
          :sampled-from="item.sampledFrom"
          class="h-12"
        />
      </div>
    </section>

    <p class="text-base-content/40 mt-4 text-xs" data-live-note>
      {{ $t("cloud-rail.metrics-live", { n: liveMinutes(series) }) }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import type { BarDataPoint } from "@/components/ui/BarChart.vue";
import { liveMinutes, type LiveSeries } from "@/composable/cloud/liveMetrics";

const { series, named = false } = defineProps<{
  series: LiveSeries[];
  /** Heads each series with its name, for a list of hosts. */
  named?: boolean;
}>();

const { t } = useI18n();

const pct = (v: number) => `${v.toFixed(1)}%`;
const bytes = (v: number) => formatBytes(v, { short: true, decimals: 1 });
const avg = (xs: number[]) => xs.reduce((a, b) => a + b, 0) / Math.max(1, xs.length);

const charts = computed(() =>
  series.map((s) => {
    // Padding is not a measurement, so it never reaches a peak or an average.
    const real = s.points.slice(s.sampledFrom);
    const cpu = real.map((p) => p.cpu);
    const memory = real.map((p) => p.memoryUsage);
    const limit = s.memoryLimit;
    const charts: {
      key: string;
      label: string;
      colorClass: string;
      data: BarDataPoint[];
      ceiling?: number;
      peak: string;
      summary: string;
      limit?: string;
    }[] = [
      {
        key: "cpu",
        label: t("label.cpu"),
        colorClass: "text-primary",
        data: s.points.map((p) => ({ percent: p.cpu, value: p.cpu })),
        peak: pct(Math.max(0, ...cpu)),
        summary: `${t("analytics.avg")} ${pct(avg(cpu))}`,
      },
      {
        key: "memory",
        label: t("label.mem"),
        colorClass: "text-secondary",
        data: s.points.map((p) => ({
          percent: limit ? (p.memoryUsage / limit) * 100 : p.memoryUsage,
          value: p.memoryUsage,
        })),
        ceiling: limit ? 100 : undefined,
        peak: bytes(Math.max(0, ...memory)),
        summary: `${t("analytics.avg")} ${bytes(avg(memory))}`,
        limit: limit ? bytes(limit) : undefined,
      },
    ];
    return { id: s.id, name: s.name, sampledFrom: s.sampledFrom, charts };
  }),
);
</script>
