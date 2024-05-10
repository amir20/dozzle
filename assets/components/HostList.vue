<template>
  <ul class="grid grid-cols-[repeat(auto-fill,minmax(480px,1fr))] gap-4">
    <li v-for="host in hostSummaries" class="card bg-base-lighter">
      <div class="card-body grid auto-cols-auto grid-flow-col justify-between">
        <div class="overflow-hidden">
          <div class="truncate text-xl font-semibold">{{ host.name }}</div>
          <ul class="flex flex-row gap-3 text-sm">
            <li><ph:cpu class="inline-block" /> {{ host.nCPU }} CPUs</li>
            <li><ph:memory class="inline-block" /> {{ formatBytes(host.memTotal) }} total</li>
          </ul>
          <div><octicon:container-24 class="inline-block" /> {{ host.containers.length }} containers</div>
        </div>

        <div class="flex flex-row gap-8">
          <div
            class="radial-progress text-primary"
            :style="`--value: ${Math.floor((host.totalCPU / (host.nCPU * 100)) * 100)}; --thickness: 0.25em`"
            role="progressbar"
          >
            {{ host.totalCPU.toFixed(0) }}%
          </div>
          <div
            class="radial-progress text-primary"
            :style="`--value: ${(host.totalMem / host.memTotal) * 100}; --thickness: 0.25em`"
            role="progressbar"
          >
            {{ formatBytes(host.totalMem, 1) }}
          </div>
        </div>
      </div>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";

const { containers } = defineProps<{
  containers: Container[];
}>();

const { hosts } = useHosts();
type HostSummary = {
  name: string;
  containers: Container[];
  totalCPU: number;
  totalMem: number;
  nCPU: number;
  memTotal: number;
};

const hostSummaries = computed(() => {
  const summaries: Record<string, HostSummary> = {};
  for (const container of containers) {
    if (!summaries[container.host]) {
      const host = hosts.value[container.host];
      summaries[container.host] = reactive({
        name: host.name,
        containers: [],
        totalCPU: 0,
        totalMem: 0,
        nCPU: host.nCPU,
        memTotal: host.memTotal,
      });
    }
    const summary = summaries[container.host];
    summary.containers.push(container);
  }

  return Object.values(summaries).sort((a, b) => a.name.localeCompare(b.name));
});

useIntervalFn(
  () => {
    for (const summary of hostSummaries.value) {
      summary.totalCPU = 0;
      summary.totalMem = 0;
      for (const container of summary.containers) {
        summary.totalCPU += container.stat.cpu;
        summary.totalMem += container.stat.memoryUsage;
      }
    }
  },
  1000,
  { immediate: true },
);
</script>

<style scoped></style>
