<template>
  <div class="card bg-base-100">
    <div class="card-body flex gap-3 max-md:p-4">
      <div class="flex flex-row items-center gap-3 overflow-hidden">
        <div class="flex min-w-0 items-center gap-2 text-lg font-semibold tracking-tight md:text-xl">
          <HostIcon :type="host.type" class="text-base-content/80 flex-none" />
          <div class="truncate">{{ host.name }}</div>

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

        <ul
          class="text-base-content/60 ml-auto flex shrink-0 flex-row flex-wrap items-center gap-x-3 text-xs tabular-nums md:text-sm"
        >
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

      <div class="grid grid-cols-2 gap-2 md:gap-3" v-if="stats">
        <MetricCard
          :icon="PhCpu"
          :value="
            cpuDisplayValue(stats.weighted.movingAverage.totalCPU, stats.weighted.movingAverage.totalCPU * cpuScale)
          "
          :chartData="cpuHistory"
          container-class="bg-primary/10"
          text-class="text-primary"
          bar-class="bg-primary"
          :formatValue="(value) => `${value.toFixed(1)}%`"
          :label="`${host.nCPU} CPU`"
        />

        <MetricCard
          :icon="PhMemory"
          :value="stats.weighted.movingAverage.totalMemUsage"
          :chartData="memHistory"
          container-class="bg-secondary/10"
          text-class="text-secondary"
          bar-class="bg-secondary"
          :formatValue="(value) => formatBytes(value, { decimals: 1 })"
          :label="formatBytes(host.memTotal, { decimals: 1 })"
        />
      </div>
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

// Scale factor to convert whole-CPU utilization into the per-core ("cores") display.
const cpuScale = computed(() => {
  let whole = 0;
  let raw = 0;
  for (const container of hostContainers.value) {
    const cpu = Math.max(0, container.stat.cpu);
    whole += cpu / toContainerCores(container);
    raw += cpu;
  }
  return whole > 0 ? raw / whole : props.host.nCPU || 1;
});

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
