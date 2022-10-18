<template>
  <svg :width="width" :height="height" @mousemove="onMove">
    <path :d="path" class="area" />
    <line :x1="lineX" y1="0" :x2="lineX" :y2="height" class="line" />
  </svg>
</template>

<script lang="ts" setup>
import { extent } from "d3-array";
import { scaleLinear } from "d3-scale";
import { area, curveStep } from "d3-shape";

const d3 = { extent, scaleLinear, area, curveStep };
const { data, width = 150, height = 30 } = defineProps<{ data: Point<unknown>[]; width?: number; height?: number }>();
const x = d3.scaleLinear().range([width, 0]);
const y = d3.scaleLinear().range([height, 0]);

const emit = defineEmits<{
  (event: "selected-point", value: Point<unknown>): void;
}>();

const shape = d3
  .area<Point<unknown>>()
  .curve(d3.curveStep)
  .x((d) => x(d.x))
  .y0(height)
  .y1((d) => y(d.y));

const path = computed(() => {
  x.domain(d3.extent(data, (d) => d.x) as [number, number]);
  y.domain(d3.extent(data, (d) => d.y) as [number, number]);

  return shape(data) ?? "";
});

let lineX = $ref(0);

function onMove(e: MouseEvent) {
  const { offsetX } = e;
  const xValue = x.invert(offsetX);
  const index = Math.round(xValue);
  lineX = x(index);
  const point = data[index];
  emit("selected-point", point);
}
</script>

<style scoped>
:deep(.area) {
  fill: var(--primary-color);
}

:deep(.line) {
  stroke: var(--secondary-color);
  stroke-width: 2;
  display: none;
}

svg:hover :deep(.line) {
  display: unset;
}
</style>
