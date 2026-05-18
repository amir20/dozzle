<template>
  <div
    class="relative max-md:hidden"
    :class="textClass"
    :title="`${titlePrefix} ↑ ${formatBytes(up)}/s · ↓ ${formatBytes(down)}/s`"
    @mouseenter="mouseOver = true"
    @mouseleave="
      mouseOver = false;
      hoveredValue = null;
    "
  >
    <div class="overflow-hidden rounded-xs border px-px pt-1 pb-px" :class="containerClass">
      <BarChart
        :chart-data="data"
        :bar-class="`${barClass} opacity-70 hover:opacity-100`"
        class="h-8 w-32"
        @hover-value="(value: number) => (hoveredValue = value)"
      />
    </div>
    <div class="bg-base-200 absolute -top-2 -left-0.5 flex items-center gap-1 rounded-sm p-px text-xs">
      <component :is="icon" class="text-sm" />
      <div class="font-bold tabular-nums select-none">{{ displayValue }}</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Component } from "vue";
import BarChart, { type BarDataPoint } from "@/components/BarChart.vue";

const { up, down } = defineProps<{
  icon: Component;
  data: BarDataPoint[];
  up: number;
  down: number;
  titlePrefix: string;
  containerClass?: string;
  textClass?: string;
  barClass?: string;
}>();

const mouseOver = ref(false);
const hoveredValue = ref<number | null>(null);

const displayValue = computed(() => {
  const value = mouseOver.value && hoveredValue.value !== null ? hoveredValue.value : up + down;
  return `${formatBytes(value, { short: true, decimals: 1 })}/s`;
});
</script>
