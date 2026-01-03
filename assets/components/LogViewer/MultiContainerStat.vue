<template>
  <div class="flex gap-1 md:gap-4">
    <StatMonitor
      :data="cpuData"
      :icon="PhCpu"
      :stat-value="Math.max(0, totalStat.cpu).toFixed(2) + '%'"
      :limit="roundCPU(limits.cpu) + ' CPU'"
      container-class="border-primary/30 bg-primary/10"
      bar-class="bg-primary"
    />
    <StatMonitor
      :data="memoryData"
      :icon="PhMemory"
      :stat-value="formatBytes(totalStat.memoryUsage)"
      :limit="formatBytes(limits.memory, { short: true, decimals: 1 })"
      container-class="border-secondary/30 bg-secondary/10"
      bar-class="bg-secondary"
    />
  </div>
</template>

<script lang="ts" setup>
import { Stat } from "@/models/Container";
import { Container } from "@/models/Container";
// @ts-ignore
import PhCpu from "~icons/ph/cpu";
// @ts-ignore
import PhMemory from "~icons/ph/memory";

const { containers } = defineProps<{
  containers: Container[];
}>();

const totalStat = ref<Stat>({ cpu: 0, memory: 0, memoryUsage: 0 });
const { history, reset } = useSimpleRefHistory(totalStat, { capacity: 300 });
const { hosts } = useHosts();

const roundCPU = (num: number) => (Number.isInteger(num) ? num.toFixed(0) : num.toFixed(1));

function toContainerCores(container: Container): number {
  if (container.cpuLimit && container.cpuLimit > 0) {
    return 1;
  }
  const hostInfo = hosts.value[container.host];
  return hostInfo?.nCPU ?? 1;
}

watch(
  () => containers,
  () => {
    const initial: Stat[] = [];
    for (let i = 1; i <= 300; i++) {
      const stat = containers.reduce(
        (acc, container) => {
          const item = container.statsHistory.at(-i);
          if (!item) {
            return acc;
          }
          const cores = toContainerCores(container);
          return {
            cpu: acc.cpu + item.cpu / cores,
            memory: acc.memory + item.memory,
            memoryUsage: acc.memoryUsage + item.memoryUsage,
          };
        },
        { cpu: 0, memory: 0, memoryUsage: 0 },
      );
      initial.push(stat);
    }
    reset({ initial: initial.reverse() });
  },
  { immediate: true },
);

const limits = computed(() => {
  return containers.reduce(
    (acc, container) => {
      const cores = toContainerCores(container);
      const hostInfo = hosts.value[container.host];

      return {
        cpu: acc.cpu + cores,
        memory: acc.memory + (container.memoryLimit || hostInfo?.memTotal || 0),
      };
    },
    { cpu: 0, memory: 0 },
  );
});

useIntervalFn(() => {
  totalStat.value = containers.reduce(
    (acc, container) => {
      const cores = toContainerCores(container);
      return {
        cpu: acc.cpu + container.stat.cpu / cores,
        memory: acc.memory + container.stat.memory,
        memoryUsage: acc.memoryUsage + container.stat.memoryUsage,
      };
    },
    { cpu: 0, memory: 0, memoryUsage: 0 },
  );
}, 1000);

const cpuData = computed(() =>
  history.value.map((stat, i) => ({
    x: i,
    y: Math.max(0, stat.cpu),
    value: Math.max(0, stat.cpu).toFixed(2) + "%",
  })),
);

const memoryData = computed(() =>
  history.value.map((stat, i) => ({
    x: i,
    y: stat.memory,
    value: formatBytes(stat.memoryUsage),
  })),
);
</script>
