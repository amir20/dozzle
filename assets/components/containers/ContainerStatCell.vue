<template>
  <div v-if="!isRunning" class="text-base-content/40 text-sm">&mdash;</div>
  <div v-else-if="isMobile" class="flex w-fit items-center gap-1.5 px-2.5 py-1 tabular-nums">
    <component :is="type === 'cpu' ? PhCpu : PhMemory" class="text-base-content/40 size-3.5 shrink-0" />
    <span class="text-[13px] font-semibold">{{ displayValue }}</span>
  </div>
  <div v-else class="flex flex-row items-center gap-2">
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
import PhCpu from "~icons/ph/cpu";
import PhMemory from "~icons/ph/memory";

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

const isRunning = computed(() => container.state === "running");

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
