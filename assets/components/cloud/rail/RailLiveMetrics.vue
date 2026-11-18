<template>
  <!--
    What the metrics panel shows while Cloud has too little history to draw:
    the live window Dozzle already holds, in the same charts, with a line that
    says where it came from so it is never mistaken for Cloud's.
  -->
  <div data-live>
    <!-- One pointer per series, as in the history it stands in for: hovering
         any of its charts marks the same sample on both. -->
    <section v-for="item in charts" :key="item.id" class="mb-6 last:mb-0" :data-series="item.id">
      <h3 v-if="named" class="mb-2 truncate font-mono text-sm font-semibold">{{ item.name }}</h3>
      <div v-for="chart in item.charts" :key="chart.key" class="mb-3 last:mb-0">
        <div class="mb-1 flex items-baseline justify-between gap-2">
          <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ chart.label }}</span>
          <span class="truncate font-mono text-xs tabular-nums">
            <span class="font-semibold">{{ readout(item, chart).main }}</span>
            <span class="text-base-content/40">{{ readout(item, chart).aside }}</span>
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
          :hover-index="hovered?.id === item.id ? hovered.index : null"
          :max="chart.ceiling"
          :sampled-from="item.sampledFrom"
          class="h-12"
          @hover-value="(_: number, index: number, bars: number) => (hovered = { id: item.id, index, bars })"
          @hover-end="hovered = null"
        />
      </div>

      <!-- Where the window starts and ends, so its span reads without a hover,
           and the hovered moment between them, so nothing below moves when
           the pointer comes and goes. -->
      <div class="text-base-content/40 mt-2 flex justify-between gap-2 font-mono text-xs tabular-nums" data-range>
        <span>{{ clock.format(new Date(item.range.from)) }}</span>
        <span v-if="hovered?.id === item.id" class="text-base-content/70 truncate" data-moment>{{
          timeOf(item, hovered)
        }}</span>
        <span>{{ clock.format(new Date(item.range.to)) }}</span>
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
import { hoverSpan, spanMiddle, useMetricsClock, type MetricsHover } from "@/composable/cloud/metricsTime";

const { series, named = false } = defineProps<{
  series: LiveSeries[];
  /** Heads each series with its name, for a list of hosts. */
  named?: boolean;
}>();

const { t } = useI18n();

// Samples arrive once a second, newest last.
const SAMPLE_MS = 1000;

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
type Item = {
  id: string;
  name: string;
  sampledFrom: number;
  length: number;
  charts: Chart[];
  // When the first and the newest sample were taken, in ms.
  range: { from: number; to: number };
};

// The bar under the pointer, and the series whose charts it is on.
const hovered = ref<(MetricsHover & { id: string }) | null>(null);

const charts = computed<Item[]>(() => {
  // Recomputed as each sample lands, so the newest bar is always now.
  const now = Date.now();
  const peakWord = t("cloud-rail.metrics-peak");
  return series.map((s) => {
    const cpuAll = s.points.map((p) => p.cpu);
    const memoryAll = s.points.map((p) => p.memoryUsage);
    // Padding is not a measurement, so it never reaches a peak or an average.
    const cpu = cpuAll.slice(s.sampledFrom);
    const memory = memoryAll.slice(s.sampledFrom);
    const real: [number, number] = [0, cpu.length];
    // A hovered bar's own run, without the padding BarChart leaves out of it too.
    const sampled = ([from, to]: [number, number]): [number, number] => [Math.max(from, s.sampledFrom), to];
    const limit = s.memoryLimit;
    const charts: Chart[] = [
      {
        key: "cpu",
        label: t("label.cpu"),
        colorClass: "text-primary",
        data: s.points.map((p) => ({ percent: p.cpu, value: p.cpu })),
        summary: `${t("analytics.avg")} ${pct(mean(cpu, real))}`,
        rest: { main: pct(Math.max(0, ...cpu)), aside: ` / ${peakWord}` },
        at: (span) => ({ main: pct(mean(cpuAll, sampled(span))), aside: "" }),
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
        summary: `${t("analytics.avg")} ${bytes(mean(memory, real))}`,
        limit: limit ? bytes(limit) : undefined,
        rest: { main: bytes(Math.max(0, ...memory)), aside: ` / ${peakWord}` },
        at: (span) => ({ main: bytes(mean(memoryAll, sampled(span))), aside: "" }),
      },
    ];
    const from = now - Math.max(0, s.points.length - 1) * SAMPLE_MS;
    return {
      id: s.id,
      name: s.name,
      sampledFrom: s.sampledFrom,
      length: s.points.length,
      charts,
      range: { from, to: now },
    };
  });
});

const readout = (item: Item, chart: Chart) =>
  hovered.value?.id === item.id ? chart.at(hoverSpan(item.length, hovered.value)) : chart.rest;

// Minutes of samples, so the seconds are what tell two bars apart.
const clock = useMetricsClock({ hour: "2-digit", minute: "2-digit", second: "2-digit" });

// A bar covers a run of samples; its middle is the honest moment to name.
function timeOf(item: Item, hover: MetricsHover) {
  const index = spanMiddle(hoverSpan(item.length, hover));
  return clock.value.format(new Date(item.range.from + index * SAMPLE_MS));
}
</script>
