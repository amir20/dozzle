<template>
  <div
    class="border-base-content/10 bg-base-100 hover:border-base-content/20 rounded-box flex flex-col gap-4 border p-4 transition-colors lg:flex-row lg:items-center lg:gap-6 lg:px-5"
  >
    <div class="flex min-w-0 flex-col gap-1 lg:shrink-0">
      <div class="flex min-w-0 items-center gap-2">
        <span class="bg-base-content/5 text-base-content/70 flex-none rounded-md p-1.5">
          <HostIcon :type="host.type" class="size-4" />
        </span>
        <div class="truncate text-lg font-semibold tracking-tight">{{ host.name }}</div>

        <span class="badge badge-error badge-xs gap-1 p-2 font-normal" v-if="!host.available">
          <carbon:warning />
          offline
        </span>
        <span
          class="badge badge-success badge-xs gap-1 p-2 font-normal"
          :class="{ 'badge-warning': config.version != host.agentVersion }"
          v-else-if="host.type == 'agent'"
          title="Dozzle Agent"
        >
          {{ host.agentVersion }}
        </span>
      </div>

      <ul class="text-base-content/50 flex flex-row flex-wrap items-center gap-x-3 gap-y-1 text-xs tabular-nums">
        <li class="flex items-center gap-1.5">
          <octicon:container-24 class="size-3.5" />
          {{ $t("label.container", hostContainers.length) }}
        </li>
        <li class="flex items-center gap-1.5" :title="runtimeLabel">
          <simple-icons:podman v-if="host.runtime === 'podman'" class="size-3.5" />
          <mdi:docker v-else class="size-3.5" />
          {{ host.dockerVersion }}
        </li>
      </ul>
    </div>

    <div class="grid grid-cols-2 gap-3 lg:flex-1" v-if="stats">
      <MetricCard
        :icon="PhCpu"
        label="CPU"
        :capacity="$t('label.core', host.nCPU ?? 0)"
        :value="stats.weighted.movingAverage.totalCPU"
        :chartData="cpuHistory"
        text-class="text-primary"
        bar-class="bg-primary"
        :formatValue="(value) => `${value.toFixed(1)}%`"
      />

      <MetricCard
        :icon="PhMemory"
        label="Memory"
        :capacity="formatBytes(host.memTotal, { decimals: 1 })"
        :value="stats.weighted.movingAverage.totalMemUsage"
        :chartData="memHistory"
        text-class="text-secondary"
        bar-class="bg-secondary"
        :formatValue="(value) => formatBytes(value, { decimals: 1 })"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Host } from "@/stores/hosts";
import { Container } from "@/models/Container";
import PhCpu from "~icons/ph/cpu";
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

const runtimeLabel = computed(() => (props.host.runtime === "podman" ? "Podman" : "Docker"));

function toContainerCores(container: Container): number {
  if (container.cpuLimit && container.cpuLimit > 0) {
    return container.cpuLimit;
  }
  return props.host.nCPU ?? 1;
}

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
        (acc, container) => {
          const item = container.statsHistory.at(-i);
          if (!item) {
            return acc;
          }
          const cores = toContainerCores(container);
          return {
            totalCPU: acc.totalCPU + item.cpu / cores,
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
    (acc, container) => {
      const cores = toContainerCores(container);
      return {
        totalCPU: acc.totalCPU + container.stat.cpu / cores,
        totalMem: acc.totalMem + container.stat.memory,
        totalMemUsage: acc.totalMemUsage + container.stat.memoryUsage,
      };
    },
    { totalCPU: 0, totalMem: 0, totalMemUsage: 0 },
  );
}, 1000);
</script>
