<template>
  <div class="card bg-base-100">
    <div class="card-body flex gap-2">
      <div class="flex flex-row gap-2 overflow-hidden">
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
        <!-- <ul class="flex flex-row gap-x-2 text-sm md:gap-3">
          <li class="flex items-center gap-1"><ph:cpu /> {{ host.nCPU }} <span class="max-md:hidden">CPUs</span></li>
          <li class="flex items-center gap-1">
            <ph:memory /> {{ formatBytes(host.memTotal) }}
            <span class="max-md:hidden">total</span>
          </li>
        </ul> -->
        <ul class="ml-auto flex flex-row flex-wrap gap-x-2 text-sm md:gap-3">
          <li class="flex items-center gap-1">
            <octicon:container-24 class="inline-block" />
            {{ $t("label.container", hostContainers.length) }}
          </li>
          <li class="flex items-center gap-1"><mdi:docker class="inline-block" /> {{ host.dockerVersion }}</li>
        </ul>
      </div>

      <div class="grid grid-cols-2 gap-2 md:gap-3" v-if="stats">
        <MetricCard
          label="CPU"
          :icon="PhCpu"
          :value="stats.weighted.movingAverage.totalCPU"
          :chartData="cpuHistory"
          container-class="border-primary/30 bg-primary/10"
          text-class="text-primary"
          bar-class="bg-primary/50"
          :formatValue="(value) => `${value.toFixed(1)}%`"
        />

        <MetricCard
          label="MEM"
          :icon="PhMemory"
          :value="stats.weighted.movingAverage.totalMemUsage"
          :chartData="memHistory"
          container-class="border-secondary/30 bg-secondary/10"
          text-class="text-secondary"
          bar-class="bg-secondary/50"
          :formatValue="formatBytes"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Host } from "@/stores/hosts";
import { Container } from "@/models/Container";
// @ts-ignore
import PhCpu from "~icons/ph/cpu";
// @ts-ignore
import PhMemory from "~icons/ph/memory";

const props = defineProps<{
  host: Host;
}>();

const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
};

const hostContainers = computed(() =>
  containers.value.filter((container) => container.host === props.host.id && container.state === "running"),
);

type TotalStat = {
  totalCPU: number;
  totalMem: number;
  totalMemUsage: number;
};

const totalStat = ref<TotalStat>({ totalCPU: 0, totalMem: 0, totalMemUsage: 0 });
const { history, reset } = useSimpleRefHistory(totalStat, { capacity: 300 });

const cpuHistory = computed(() =>
  history.value.map((stat) => ({
    percent: stat.totalCPU,
    value: stat.totalCPU,
  })),
);
const memHistory = computed(() =>
  history.value.map((stat) => ({
    percent: stat.totalMem,
    value: stat.totalMemUsage,
  })),
);

const stats = reactive({ mostRecent: totalStat, weighted: useExponentialMovingAverage(totalStat) });

watch(
  () => hostContainers.value,
  () => {
    const initial: TotalStat[] = [];
    for (let i = 1; i <= 300; i++) {
      const stat = hostContainers.value.reduce(
        (acc, { statsHistory }) => {
          const item = statsHistory.at(-i);
          if (!item) {
            return acc;
          }
          return {
            totalCPU: acc.totalCPU + item.cpu,
            totalMem: acc.totalMem + item.memory,
            totalMemUsage: acc.totalMemUsage + item.memoryUsage,
          };
        },
        { totalCPU: 0, totalMem: 0, totalMemUsage: 0 },
      );
      initial.push(stat);
    }
    reset({ initial: initial.reverse() });
    stats.weighted.reset(initial.at(-1)!);
  },
  { immediate: true },
);

useIntervalFn(() => {
  totalStat.value = hostContainers.value.reduce(
    (acc, { stat }) => {
      return {
        totalCPU: acc.totalCPU + stat.cpu,
        totalMem: acc.totalMem + stat.memory,
        totalMemUsage: acc.totalMemUsage + stat.memoryUsage,
      };
    },
    { totalCPU: 0, totalMem: 0, totalMemUsage: 0 },
  );
}, 1000);
</script>
