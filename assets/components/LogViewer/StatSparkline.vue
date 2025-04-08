<template>
  <svg :width="width" :height="height" @mousemove="onMove" class="group">
    <path :d="path" class="fill-primary" />
    <line :x1="lineX" y1="0" :x2="lineX" :y2="height" class="stroke-secondary invisible stroke-2 group-hover:visible" />
  </svg>
</template>

<script lang="ts" setup>
import { extent } from "d3-array";
import { scaleLinear } from "d3-scale";
import { area, curveStep } from "d3-shape";

const d3 = { extent, scaleLinear, area, curveStep };
const { data, width = 175, height = 30 } = defineProps<{ data: Point<unknown>[]; width?: number; height?: number }>();
const x = d3.scaleLinear().range([0, width]);
const y = d3.scaleLinear().range([height, 0]);

const selectedPoint = defineEmit<[value: Point<unknown>]>();

const shape = d3
  .area<Point<unknown>>()
  .curve(d3.curveStep)
  .x((d) => x(d.x))
  .y0(height)
  .y1((d) => y(d.y));

const path = computed(() => {
  x.domain(d3.extent(data, (d) => d.x) as [number, number]);
  y.domain(d3.extent([...data, { y: 1 }], (d) => d.y) as [number, number]);

  return shape(data) ?? "";
});

let lineX = $ref(0);

function onMove(e: MouseEvent) {
  const { offsetX } = e;
  const xValue = x.invert(offsetX);
  const index = Math.round(xValue);
  lineX = x(index);
  const point = data[index];
  selectedPoint(point);
}
</script>
