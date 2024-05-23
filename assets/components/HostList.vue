<template>
  <ul class="grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(480px,1fr))]">
    <li v-for="host in hosts" class="card bg-base-lighter">
      <div class="card-body grid auto-cols-auto grid-flow-col justify-between gap-4">
        <div class="overflow-hidden">
          <div class="truncate text-xl font-semibold">
            {{ host.name }} <span class="badge badge-error badge-xs p-1.5" v-if="!host.available">offline</span>
          </div>
          <ul class="flex flex-row gap-2 text-sm md:gap-4">
            <li><ph:cpu class="inline-block" /> {{ host.nCPU }} <span class="mobile-hidden">CPUs</span></li>
            <li>
              <ph:memory class="inline-block" /> {{ formatBytes(host.memTotal) }}
              <span class="mobile-hidden">total</span>
            </li>
          </ul>
          <div class="text-sm">
            <octicon:container-24 class="inline-block" /> {{ $t("label.container", hostContainers[host.id]?.length) }}
          </div>
        </div>

        <div class="flex flex-row gap-4 md:gap-8" v-if="weightedStats[host.id]">
          <div
            class="radial-progress text-sm text-primary [--size:4rem] [--thickness:0.25em] md:text-[1rem] md:[--size:5rem]"
            :style="`--value: ${Math.floor((weightedStats[host.id].weighted.totalCPU / (host.nCPU * 100)) * 100)};  `"
            role="progressbar"
          >
            {{ weightedStats[host.id].weighted.totalCPU.toFixed(0) }}%
          </div>
          <div
            class="radial-progress text-sm text-primary [--size:4rem] [--thickness:0.25em] md:text-[1rem] md:[--size:5rem]"
            :style="`--value: ${(weightedStats[host.id].weighted.totalMem / host.memTotal) * 100};`"
            role="progressbar"
          >
            {{ formatBytes(weightedStats[host.id].weighted.totalMem, 1) }}
          </div>
        </div>
      </div>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";

const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
};

const runningContainers = computed(() => containers.value.filter((container) => container.state === "running"));

const { hosts } = useHosts();
const hostContainers = computed(() => {
  const results: Record<string, Container[]> = {};
  for (const container of runningContainers.value) {
    if (!results[container.host]) {
      results[container.host] = [];
    }
    results[container.host].push(container);
  }
  return results;
});

type TotalStat = {
  totalCPU: number;
  totalMem: number;
};
const weightedStats: Record<string, { mostRecent: TotalStat; weighted: TotalStat }> = {};
const initWeightedStats = () => {
  for (const [host, containers] of Object.entries(hostContainers.value)) {
    const mostRecent = ref<TotalStat>({ totalCPU: 0, totalMem: 0 });
    for (const container of containers) {
      mostRecent.value.totalCPU += container.stat.cpu;
      mostRecent.value.totalMem += container.stat.memoryUsage;
    }
    weightedStats[host] = reactive({ mostRecent, weighted: useExponentialMovingAverage(mostRecent) });
  }
};

watchOnce(hostContainers, initWeightedStats);
initWeightedStats();

useIntervalFn(
  () => {
    for (const [host, containers] of Object.entries(hostContainers.value)) {
      const stat = { totalCPU: 0, totalMem: 0 };
      for (const container of containers) {
        stat.totalCPU += container.stat.cpu;
        stat.totalMem += container.stat.memoryUsage;
      }
      weightedStats[host].mostRecent = stat;
    }
  },
  1000,
  { immediate: true },
);
</script>

<style scoped></style>
