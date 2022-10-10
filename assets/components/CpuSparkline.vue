<template>
  <svg width="140" height="40"></svg>
</template>

<script lang="ts" setup>
import { select, ValueFn } from "d3-selection";
import { extent } from "d3-array";
import { scaleLinear } from "d3-scale";
import { area } from "d3-shape";
import { Container } from "@/models/Container";
import { ComputedRef } from "vue";

const d3 = { select, extent, scaleLinear, area };

const container = inject("container") as ComputedRef<Container>;

const root = useCurrentElement();

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
    .x((d: any) => x(d.x))
    .y0(y(0))
    .y1((d: any) => y(d.y)) as ValueFn<SVGGElement, any, string>;

  function tick() {
    const data = cpuData();

    x.domain(d3.extent(data, (d) => d.x) as [number, number]);
    y.domain(d3.extent(data, (d) => d.y) as [number, number]);

    path.datum(data).attr("d", area);
  }

  watch(() => container.value.stat, tick);
});

const cpuData = () => {
  const history = container.value.getStatHistory();
  return history.map((stat, i) => ({ x: history.length - i, y: stat.snapshot.cpu }));
};
</script>

<style scoped>
:deep(.area) {
  fill: steelblue;
  stroke: steelblue;
  stroke-width: 1;
}
</style>
