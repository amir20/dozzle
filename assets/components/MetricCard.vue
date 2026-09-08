<template>
  <div class="border-base-content/10 bg-base-200/40 rounded-lg border px-3 py-2.5">
    <div class="flex items-center gap-2">
      <div class="flex min-w-0 items-center gap-1.5 text-xs font-medium" :class="textClass">
        <component :is="icon" class="size-3.5 shrink-0" />
        <span class="truncate">{{ label }}</span>
      </div>
      <span class="text-base-content/40 ml-auto shrink-0 text-xs tabular-nums">{{ capacity }}</span>
    </div>

    <div class="mt-0.5 text-xl leading-tight font-semibold tabular-nums">{{ formattedValue }}</div>
    <div class="text-base-content/50 mt-0.5 truncate text-[0.6875rem] tabular-nums">
      avg {{ formatValue(average) }} · pk {{ formatValue(peak) }}
    </div>

    <BarChart class="mt-2 h-7" :chart-data="chartData" :bar-class="barClass" />
  </div>
</template>

<script setup lang="ts">
import type { Component } from "vue";
import type { BarDataPoint } from "@/components/BarChart.vue";

const {
  label,
  capacity,
  icon,
  value,
  chartData,
  textClass = "",
  barClass = "",
  formatValue = (v: number) => v.toString(),
} = defineProps<{
  label: string;
  capacity: string;
  icon: Component;
  value: string | number;
  chartData: BarDataPoint[];
  textClass?: string;
  barClass?: string;
  formatValue?: (value: number) => string;
}>();

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
