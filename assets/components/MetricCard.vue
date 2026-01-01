<template>
  <div class="rounded-lg border p-3" :class="containerClass">
    <div class="mb-2 flex items-center gap-1.5 text-xs font-medium" :class="textClass">
      <component :is="icon" class="text-sm" />
      <span>{{ label }}</span>
    </div>
    <div class="mb-1.5 text-lg font-semibold">{{ formattedValue }}</div>
    <div class="text-base-content/60 mb-1 text-[10px]">avg {{ formatValue(average) }} • pk {{ formatValue(peak) }}</div>
    <BarChart class="h-8" :chartData="percentData" :barClass="barClass" />
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

const percentData = computed(() => chartData.map((d) => d.percent));

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
