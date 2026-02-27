<template>
  <div
    class="relative"
    @mouseenter="mouseOver = true"
    @mouseleave="
      mouseOver = false;
      hoveredValue = null;
    "
    :class="textClass"
  >
    <div class="overflow-hidden rounded-xs border px-px pt-1 pb-px max-md:hidden" :class="containerClass">
      <BarChart
        ref="barChartRef"
        :chart-data="chartData"
        :bar-class="`${barClass} opacity-70 hover:opacity-100`"
        class="h-8 w-44"
        @hover-value="(value: number) => (hoveredValue = value)"
      />
    </div>
    <div class="bg-base-200 flex gap-1 rounded-sm p-px text-xs md:absolute md:-top-2 md:-left-0.5">
      <component :is="icon" class="text-sm" />
      <div class="font-bold tabular-nums select-none">
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
  formatter,
} = defineProps<{
  data: Point<number>[];
  icon: Component;
  statValue: string | number;
  limit?: string | number;
  containerClass?: string;
  textClass?: string;
  barClass?: string;
  formatter?: (value: number) => string;
}>();

const chartData = computed(() =>
  data.map((point) => ({
    percent: point.y ?? 0,
    value: point.value ?? point.y ?? 0,
  })),
);
const barChartRef = ref<InstanceType<typeof BarChart> | null>(null);
const mouseOver = ref(false);
const hoveredValue = ref<number | null>(null);

defineExpose({ recalculate: () => barChartRef.value?.recalculate() });

const displayValue = computed(() => {
  if (mouseOver.value && hoveredValue.value !== null) {
    if (formatter) {
      return formatter(hoveredValue.value);
    }
    return hoveredValue.value.toFixed(2);
  }
  return statValue;
});
</script>
