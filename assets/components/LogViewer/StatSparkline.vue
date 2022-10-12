<template>
  <svg width="150" height="20"></svg>
</template>

<script lang="ts" setup>
import { select, type ValueFn } from "d3-selection";
import { extent } from "d3-array";
import { scaleLinear } from "d3-scale";
import { area, curveStep } from "d3-shape";

const d3 = { select, extent, scaleLinear, area, curveStep };

const root = useCurrentElement();

const { data } = defineProps<{ data: { x: number; y: number }[] }>();

onMounted(() => {
  const svg = d3.select(root.value);
  const width = +svg.attr("width");
  const height = +svg.attr("height");
  const margin = { top: 0, right: 0, bottom: 0, left: 0 };
  const innerWidth = width - margin.left - margin.right;
  const innerHeight = height - margin.top - margin.bottom;

  const x = d3.scaleLinear().range([0, innerWidth]);
  const y = d3.scaleLinear().range([innerHeight, 0]);

  const g = svg.append("g").attr("transform", `translate(${margin.left}, ${margin.top})`);

  const path = g.append("path").attr("class", "area");

  const area = d3
    .area()
    .curve(d3.curveStep)
    .x((d: any) => x(d.x))
    .y0(y(0))
    .y1((d: any) => y(d.y)) as ValueFn<SVGGElement, any, string>;

  watchEffect(() => {
    x.domain(d3.extent(data, (d) => d.x) as [number, number]);
    y.domain(d3.extent(data, (d) => d.y) as [number, number]);

    path.datum(data).attr("d", area);
  });
});
</script>

<style scoped>
:deep(.area) {
  fill: var(--primary-color);
  stroke: var(--primary-color);
  stroke-width: 1;
}
</style>
