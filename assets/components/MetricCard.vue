<template>
  <div class="rounded-lg border p-3" :class="containerClass">
    <div class="mb-2 flex items-center gap-1.5 text-xs font-medium" :class="textClass">
      <component :is="icon" class="text-sm" />
      <span>{{ label }}</span>
    </div>
    <div class="mb-1.5 text-lg font-semibold">{{ formattedValue }}</div>
    <div class="text-base-content/60 mb-1 text-[10px]">avg {{ formatValue(average) }} • pk {{ formatValue(peak) }}</div>
    <!-- Bar chart -->
    <div ref="chartContainer" class="flex h-8 items-end gap-[2px]">
      <div
        v-for="(dataPoint, i) in downsampledData"
        :key="i"
        class="flex-1 rounded-t-sm"
        :class="barClass"
        :style="`height: ${Math.min(dataPoint, 100)}%`"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from "vue";

export interface MetricDataPoint {
  percent: number; // value 0 - 100
  value: number;
}

const {
  label,
  icon,
  value,
  chartData,
  containerClass = "",
  textClass = "",
  barClass = "",
  formatValue = (v: number) => v.toString(),
} = defineProps<{
  label: string;
  icon: Component;
  value: string | number;
  chartData: MetricDataPoint[];
  containerClass?: string;
  textClass?: string;
  barClass?: string;
  formatValue?: (value: number) => string;
}>();

const chartContainer = ref<HTMLElement | null>(null);
const { width } = useElementSize(chartContainer);

const downsampledData = computed(() => {
  const BAR_WIDTH = 3;
  const GAP = 2;
  const availableBars = Math.floor(width.value / (BAR_WIDTH + GAP));

  if (chartData.length <= availableBars || availableBars === 0) {
    return chartData.map((d) => d.percent);
  }

  // Downsample by averaging buckets
  const bucketSize = chartData.length / availableBars;
  const result = [];
  for (let i = 0; i < availableBars; i++) {
    const start = Math.floor(i * bucketSize);
    const end = Math.floor((i + 1) * bucketSize);
    const bucket = chartData.slice(start, end);
    const avg = bucket.reduce((sum, val) => sum + val.percent, 0) / bucket.length;
    result.push(avg);
  }
  return result;
});

const peak = computed(() => (chartData.length > 0 ? Math.max(...chartData.map((d) => d.value)) : 0));

const average = computed(() => {
  if (chartData.length === 0) return 0;
  return chartData.reduce((sum, d) => sum + d.value, 0) / chartData.length;
});

const formattedValue = computed(() => {
  if (typeof value === "string") return value;
  return formatValue(value);
});
</script>
