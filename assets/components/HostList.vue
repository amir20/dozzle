<template>
  <ul class="grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(480px,1fr))]">
    <li v-for="host in hosts" class="card bg-base-100">
      <div class="card-body grid auto-cols-auto grid-flow-col justify-between gap-4">
        <div class="flex flex-col gap-2 overflow-hidden">
          <div class="flex items-center gap-1 truncate text-xl font-semibold">
            <HostIcon :type="host.type" class="flex-none" />
            <div class="truncate">
              {{ host.name }}
            </div>

            <span class="badge badge-error badge-xs gap-2 p-2" v-if="!host.available">
              <carbon:warning />
              offline
            </span>
            <span
              class="badge badge-success badge-xs gap-2 p-2"
              :class="{ 'badge-warning': config.version != host.agentVersion }"
              v-else-if="host.type == 'agent'"
              title="Dozzle Agent"
            >
              {{ host.agentVersion }}
            </span>
          </div>
          <ul class="flex flex-row gap-x-2 text-sm md:gap-3">
            <li class="flex items-center gap-1"><ph:cpu /> {{ host.nCPU }} <span class="max-md:hidden">CPUs</span></li>
            <li class="flex items-center gap-1">
              <ph:memory /> {{ formatBytes(host.memTotal) }}
              <span class="max-md:hidden">total</span>
            </li>
          </ul>
          <ul class="flex flex-row flex-wrap gap-x-2 text-sm md:gap-3">
            <li class="flex items-center gap-1">
              <octicon:container-24 class="inline-block" />
              {{ $t("label.container", hostContainers[host.id]?.length ?? 0) }}
            </li>
            <li class="flex items-center gap-1"><mdi:docker class="inline-block" /> {{ host.dockerVersion }}</li>
          </ul>
        </div>

        <div class="flex flex-row gap-4 md:gap-8" v-if="weightedStats[host.id]">
          <div
            class="radial-progress text-primary text-[0.85rem] transition-none [--size:4rem] [--thickness:0.25em] md:text-[0.9rem] md:[--size:5rem]"
            :style="`--value: ${Math.floor((weightedStats[host.id].weighted.totalCPU / (host.nCPU * 100)) * 100)};`"
            role="progressbar"
          >
            {{ weightedStats[host.id].weighted.totalCPU.toFixed(0) }}%
          </div>
          <div
            class="radial-progress text-primary text-[0.85rem] transition-none [--size:4rem] [--thickness:0.25em] md:text-[0.9rem] md:[--size:5rem]"
            :style="`--value: ${Math.floor((weightedStats[host.id].weighted.totalMem / host.memTotal) * 100)};`"
            role="progressbar"
          >
            {{ formatBytes(weightedStats[host.id].weighted.totalMem, { decimals: 1, short: true }) }}
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
      if (weightedStats[host]) {
        // TODO fix this init
        weightedStats[host].mostRecent = stat;
      }
    }
  },
  1000,
  { immediate: true },
);
</script>
