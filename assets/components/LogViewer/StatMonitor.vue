<template>
  <div class="relative" @mouseenter="mouseOver = true" @mouseleave="mouseOver = false" :class="textClass">
    <div class="overflow-hidden rounded-xs border px-px pt-1 pb-px max-md:hidden" :class="containerClass">
      <BarChart
        :chart-data="chartData"
        :bar-class="`${barClass} opacity-70 hover:opacity-100`"
        class="h-8 w-44"
        @hover-index="(index: number) => (hoveredIndex = index)"
      />
    </div>
    <div class="bg-base-200 inline-flex gap-1 rounded-sm p-px text-xs md:absolute md:-top-2 md:-left-0.5">
      <component :is="icon" class="text-sm" />
      <div class="font-bold select-none">
        {{ displayValue }}
        <span v-if="limit !== -1 && !mouseOver" class="max-md:hidden"> / {{ limit }} </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Component } from "vue";

const {
  data,
  icon,
  statValue,
  limit = -1,
  containerClass = "border-primary",
  textClass = "",
  barClass = "bg-primary",
} = defineProps<{
  data: Point<unknown>[];
  icon: Component;
  statValue: string | number;
  limit?: string | number;
  containerClass?: string;
  textClass?: string;
  barClass?: string;
}>();

const chartData = computed(() => data.map((point) => (point.y as number) ?? 0));
const mouseOver = ref(false);
const hoveredIndex = ref<number | null>(null);

const displayValue = computed(() => {
  if (mouseOver.value && hoveredIndex.value !== null) {
    const point = data[hoveredIndex.value];
    return point?.value ?? point?.y ?? statValue;
  }
  return statValue;
});
</script>
