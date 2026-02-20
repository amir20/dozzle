<template>
  <div class="flex flex-row items-center gap-2">
    <BarChart class="h-4 flex-1" :chart-data="chartData" :bar-class="barClass" />
    <span class="w-fit text-right text-sm">{{ displayValue }}</span>
  </div>
</template>

<script setup lang="ts">
import type { Container } from "@/models/Container";
import type { Host } from "@/stores/hosts";

const { container, type, host } = defineProps<{
  container: Container;
  type: "cpu" | "mem";
  host: Host;
}>();

function totalCores(): number {
  if (container.cpuLimit && container.cpuLimit > 0) {
    return container.cpuLimit;
  }
  return host.nCPU ?? 1;
}

const chartData = computed(() => {
  if (type === "cpu") {
    const cores = totalCores();
    return container.statsHistory.map((stat) => Math.min(stat.cpu / cores, 100));
  }
  return container.statsHistory.map((stat) => Math.min(stat.memory, 100));
});

const averageValue = computed(() => {
  if (type === "cpu") {
    const cores = totalCores();
    return Math.min(container.movingAverage.cpu / cores, 100);
  }
  return container.movingAverage.memory;
});

const displayValue = computed(() => {
  if (type === "cpu") {
    return `${averageValue.value.toFixed(0)}%`;
  }
  return formatBytes(container.movingAverage.memoryUsage);
});

const barClass = computed(() => {
  const value = averageValue.value;
  if (value <= 50) return "bg-success";
  if (value <= 70) return "bg-secondary";
  if (value <= 90) return "bg-warning";
  return "bg-error";
});
</script>
