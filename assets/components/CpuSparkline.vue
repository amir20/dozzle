<template>
  <svg width="140" height="40"></svg>
</template>

<script lang="ts" setup>
import { select, ValueFn } from "d3-selection";
import { extent } from "d3-array";
import { active, transition } from "d3-transition";
import { scaleLinear } from "d3-scale";
import { line } from "d3-shape";
import { easeLinear } from "d3-ease";
import { Container } from "@/models/Container";
import { ComputedRef } from "vue";

const d3 = { select, extent, active, transition, scaleLinear, line, easeLinear };

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

  const line = d3
    .line()
    .x((d: any) => x(d.x))
    .y((d: any) => y(d.y)) as ValueFn<any, any, string>;

  const g = svg.append("g").attr("transform", `translate(${margin.left}, ${margin.top})`);

  const path = g
    .append("path")
    .attr("fill", "none")
    .attr("stroke", "steelblue")
    .attr("stroke-width", 1)
    .attr("stroke-linejoin", "round")
    .attr("stroke-linecap", "round");

  // const data = cpuData();

  // x.domain(d3.extent(data, (d) => d.x) as [number, number]);
  // y.domain(d3.extent(data, (d) => d.y) as [number, number]);

  // path.datum(data).attr("d", line);

  // const t = d3.transition().duration(1000).ease(d3.easeLinear);

  // path.datum(data).attr("d", line);
  // .transition(t)
  // .attr("transform", "translate(" + x(-1) + ",0)")
  // .on("start", tick);

  function tick() {
    const data = cpuData();

    x.domain(d3.extent(data, (d) => d.x) as [number, number]);
    y.domain(d3.extent(data, (d) => d.y) as [number, number]);
    path.datum(data).attr("d", line);

    // path.datum(data).attr("d", line).attr("transform", null);

    // d3.active(this)
    //   .transition(t)
    //   .attr("transform", "translate(" + x(-1) + ",0)")
    //   .on("start", tick);
  }

  watch(() => container.value.stat, tick);
});

const cpuData = () => {
  const history = container.value.getStatHistory();
  return history.map((stat, i) => ({ x: history.length - i, y: stat.snapshot.cpu }));
};
</script>
