<template>
  <svg :width="width" :height="height">
    <path :d="path" class="area" />
  </svg>
</template>

<script lang="ts" setup>
import { extent } from "d3-array";
import { scaleLinear } from "d3-scale";
import { area, curveStep } from "d3-shape";

const d3 = { extent, scaleLinear, area, curveStep };
const { data, width = 150, height = 30 } = defineProps<{ data: Point[]; width?: number; height?: number }>();
const x = d3.scaleLinear().range([0, width]);
const y = d3.scaleLinear().range([height, 0]);

const shape = d3
  .area<Point>()
  .curve(d3.curveStep)
  .x((d) => x(d.x))
  .y0(height)
  .y1((d) => y(d.y));

const path = computed(() => {
  x.domain(d3.extent(data, (d) => d.x) as [number, number]);
  y.domain(d3.extent(data, (d) => d.y) as [number, number]);
  return shape(data) ?? "";
});
</script>

<style scoped>
:deep(.area) {
  fill: var(--primary-color);
}
</style>
