<template>
  <div class="flex gap-4" v-if="container.stat">
    <stat-monitor :data="memoryData" label="mem" :stat-value="formatBytes(unref(container.stat).memoryUsage)" />
    <stat-monitor :data="cpuData" label="load" :stat-value="unref(container.stat).cpu + '%'" />
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const container = inject("container") as ComputedRef<Container>;

const cpuData = computedWithControl(
  () => container.value.stat,
  () => {
    const history = container.value.statHistory;
    const points: Point<unknown>[] = history.map((stat, i) => ({
      x: i,
      y: stat.snapshot.cpu,
      value: stat.snapshot.cpu + "%",
    }));
    return points;
  },
);

const memoryData = computedWithControl(
  () => container.value.stat,
  () => {
    const history = container.value.statHistory;
    const points: Point<string>[] = history.map((stat, i) => ({
      x: i,
      y: stat.snapshot.memory,
      value: formatBytes(stat.snapshot.memoryUsage),
    }));
    return points;
  },
);
</script>

<style scoped></style>
