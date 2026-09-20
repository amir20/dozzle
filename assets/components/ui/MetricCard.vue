<template>
  <div class="border-base-content/10 bg-base-200/40 rounded-lg border px-3 py-2.5">
    <!-- Hovering the chart reads it back: a bar is a shape until it says what it was
         and when. Leaving falls back to the live value with avg/pk under it, so the
         card keeps its three lines either way and nothing reflows under the pointer.

         The comment sits inside the root: above it, it compiles to a second root node
         in dev builds and `wrapper.element` is then the comment, which every
         `trigger()` in the spec would fire at instead. -->
    <div class="flex items-center gap-2">
      <div class="flex min-w-0 items-center gap-1.5 text-xs font-medium" :class="textClass">
        <component :is="icon" class="size-3.5 shrink-0" />
        <span class="truncate">{{ label }}</span>
      </div>
      <span class="text-base-content/40 ml-auto shrink-0 text-xs tabular-nums">{{ capacity }}</span>
    </div>

    <div class="mt-0.5 text-xl leading-tight font-semibold tabular-nums">
      {{ hovered ? formatValue(hovered.value) : formattedValue }}
    </div>
    <div class="text-base-content/50 mt-0.5 truncate text-[0.6875rem] tabular-nums">
      <template v-if="hoveredTime">{{ hoveredTime }}</template>
      <template v-else>avg {{ formatValue(average) }} · pk {{ formatValue(peak) }}</template>
    </div>

    <BarChart
      ref="chart"
      class="mt-2 h-7"
      :chart-data="chartData"
      :bar-class="barClass"
      :max="chartMax"
      :sampled-from="sampledFrom"
      @hover-value="(value: number, index: number, bars: number) => (hovered = { value, index, bars })"
      @hover-end="hovered = undefined"
    />
  </div>
</template>

<script setup lang="ts">
import type { Component } from "vue";
import type { BarDataPoint } from "./BarChart.vue";

const {
  label,
  capacity,
  icon,
  value,
  chartData,
  textClass = "",
  barClass = "",
  chartMax,
  sampleInterval,
  sampledFrom = 0,
  formatValue = (v: number) => v.toString(),
} = defineProps<{
  label: string;
  capacity: string;
  icon: Component;
  value: string | number;
  chartData: BarDataPoint[];
  textClass?: string;
  barClass?: string;
  chartMax?: number;
  // Index in `chartData` where real samples begin; see BarChart's own prop.
  sampledFrom?: number;
  // Milliseconds between two points of `chartData`, newest last. Only the parent
  // knows how often it samples, and without it a hovered bar has no time to name,
  // so the readout stays on avg/pk.
  sampleInterval?: number;
  formatValue?: (value: number) => string;
}>();

const { locale } = useI18n();

const hovered = ref<{ value: number; index: number; bars: number } | undefined>();

// The chart is rendered in here rather than by the parent, so the parent cannot
// reach it to force a full re-bucket after it replaces the series wholesale. This
// forwards that call; see HostCard, which makes it when its container set changes.
const chart = useTemplateRef("chart");
defineExpose({ recalculate: () => chart.value?.recalculate() });

// Averaging the padding in was what made a fresh card read `avg 414.9 KB` beside
// `pk 7.2 MB`: the mean was divided by samples nobody took.
const sampled = computed(() => chartData.slice(sampledFrom));

const peak = computed(() => (sampled.value.length > 0 ? Math.max(...sampled.value.map((d) => d.value)) : 0));

const average = computed(() => {
  if (sampled.value.length === 0) return 0;
  return sampled.value.reduce((sum, d) => sum + d.value, 0) / sampled.value.length;
});

const formattedValue = computed(() => {
  if (typeof value === "string") return value;
  return formatValue(value);
});

// A bar is a bucket of points, so the hovered one covers a span. Its middle is the
// honest moment to name.
const hoveredTime = computed(() => {
  if (!hovered.value || !sampleInterval || !chartData.length) return "";
  const { index, bars } = hovered.value;
  if (bars === 0) return "";
  const perBar = chartData.length / bars;
  const point = Math.min(chartData.length - 1, Math.floor((index + 0.5) * perBar));
  const age = (chartData.length - 1 - point) * sampleInterval;
  return toRelativeTime(new Date(Date.now() - age), locale.value === "" ? undefined : locale.value, "short");
});
</script>
