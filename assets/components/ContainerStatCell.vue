<template>
  <div class="flex flex-row items-center gap-2">
    <template v-if="mode === 'chart'">
      <BarChart class="h-4 flex-1" :chart-data="chartData" :bar-class="barClass" />
    </template>
    <template v-else>
      <progress class="progress flex-1" :class="progressClass" :value="averageValue" max="100"></progress>
    </template>
    <span class="min-w-12 text-right text-sm tabular-nums">{{ displayValue }}</span>
  </div>
</template>

<script setup lang="ts">
import type { Container } from "@/models/Container";
import type { Host } from "@/stores/hosts";

const {
  container,
  type,
  host,
  mode = "chart",
} = defineProps<{
  container: Container;
  type: "cpu" | "mem";
  host: Host;
  mode?: "chart" | "progress";
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
    return container.statsHistory.map((stat) => {
      const percent = Math.min(stat.cpu / cores, 100);
      return { percent, value: stat.cpu };
    });
  }
  return container.statsHistory.map((stat) => {
    const percent = Math.min(stat.memory, 100);
    return { percent, value: stat.memoryUsage };
  });
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

const progressClass = computed(() => {
  const value = averageValue.value;
  if (value <= 50) return "progress-success";
  if (value <= 70) return "progress-secondary";
  if (value <= 90) return "progress-warning";
  return "progress-error";
});
</script>
