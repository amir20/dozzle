<template>
  <div class="flex gap-1 md:gap-4">
    <div
      class="grid hidden min-w-15 grid-cols-[auto_1fr_auto_1fr] items-center gap-0.5 text-xs leading-none sm:grid md:grid-cols-[auto_1fr]"
    >
      <PhArrowUp class="text-primary" />
      <span class="tabular-nums">{{ formatBytes(networkRate.tx, { short: true, decimals: 1 }) }}/s</span>
      <PhArrowDown class="text-secondary" />
      <span class="tabular-nums">{{ formatBytes(networkRate.rx, { short: true, decimals: 1 }) }}/s</span>
    </div>
    <StatMonitor
      :data="cpuData"
      :icon="PhCpu"
      :stat-value="Math.max(0, totalStat.cpu).toFixed(2) + '%'"
      :limit="roundCPU(limits.cpu) + ' CPU'"
      container-class="border-primary/40 bg-primary/20"
      text-class="hover:text-primary"
      bar-class="bg-primary"
      :formatter="(value: number) => value.toFixed(2) + '%'"
    />
    <StatMonitor
      :data="memoryData"
      :icon="PhMemory"
      :stat-value="formatBytes(totalStat.memoryUsage)"
      :limit="formatBytes(limits.memory, { short: true, decimals: 1 })"
      container-class="border-secondary/40 bg-secondary/20"
      text-class="hover:text-secondary"
      bar-class="bg-secondary"
      :formatter="(value: number) => formatBytes(value)"
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

const totalStat = ref<Stat>({ cpu: 0, memory: 0, memoryUsage: 0, networkRxTotal: 0, networkTxTotal: 0 });
const { history, reset } = useSimpleRefHistory(totalStat, { capacity: 300 });
const { hosts } = useHosts();
const networkRate = ref({ rx: 0, tx: 0 });

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
            networkRxTotal: acc.networkRxTotal + item.networkRxTotal,
            networkTxTotal: acc.networkTxTotal + item.networkTxTotal,
          };
        },
        { cpu: 0, memory: 0, memoryUsage: 0, networkRxTotal: 0, networkTxTotal: 0 },
      );
      initial.push(stat);
    }
    totalStat.value = initial[0];
    reset({ initial: initial.reverse() });
  },
  { immediate: true },
);

const limits = computed(() => {
  // Group containers by host
  const containersByHost = new Map<string, Container[]>();
  containers.forEach((container) => {
    if (!containersByHost.has(container.host)) {
      containersByHost.set(container.host, []);
    }
    containersByHost.get(container.host)!.push(container);
  });

  let totalCpu = 0;
  let totalMemory = 0;

  // Process each host independently
  containersByHost.forEach((hostContainers, hostId) => {
    const hostInfo = hosts.value[hostId];
    const hostTotalMemory = hostInfo?.memTotal || 0;
    const hostTotalCpu = hostInfo?.nCPU || 0;

    // Check if any container lacks limits
    const hasUnlimitedCpu = hostContainers.some((c) => !c.cpuLimit || c.cpuLimit <= 0);
    const hasUnlimitedMemory = hostContainers.some((c) => !c.memoryLimit);

    // Calculate CPU for this host
    if (hasUnlimitedCpu) {
      // At least one container has no limit, use host total
      totalCpu += hostTotalCpu;
    } else {
      // All containers have limits, sum them up (capped at host total)
      const sumCpu = hostContainers.reduce((sum, c) => sum + (c.cpuLimit || 0), 0);
      totalCpu += Math.min(sumCpu, hostTotalCpu);
    }

    // Calculate Memory for this host
    if (hasUnlimitedMemory) {
      // At least one container has no limit, use host total
      totalMemory += hostTotalMemory;
    } else {
      // All containers have limits, sum them up (capped at host total)
      const sumMemory = hostContainers.reduce((sum, c) => sum + (c.memoryLimit || 0), 0);
      totalMemory += Math.min(sumMemory, hostTotalMemory);
    }
  });

  return {
    cpu: totalCpu,
    memory: totalMemory,
  };
});

useIntervalFn(() => {
  const previousStat = totalStat.value;
  totalStat.value = containers.reduce(
    (acc, container) => {
      const cores = toContainerCores(container);
      return {
        cpu: acc.cpu + container.stat.cpu / cores,
        memory: acc.memory + container.stat.memory,
        memoryUsage: acc.memoryUsage + container.stat.memoryUsage,
        networkRxTotal: acc.networkRxTotal + container.stat.networkRxTotal,
        networkTxTotal: acc.networkTxTotal + container.stat.networkTxTotal,
      };
    },
    { cpu: 0, memory: 0, memoryUsage: 0, networkRxTotal: 0, networkTxTotal: 0 },
  );

  networkRate.value = {
    rx: Math.max(0, totalStat.value.networkRxTotal - previousStat.networkRxTotal),
    tx: Math.max(0, totalStat.value.networkTxTotal - previousStat.networkTxTotal),
  };
}, 1000);

const cpuData = computed(() =>
  history.value.map((stat, i) => ({
    x: i,
    y: Math.max(0, stat.cpu),
    value: Math.max(0, stat.cpu),
  })),
);

const memoryData = computed(() =>
  history.value.map((stat, i) => ({
    x: i,
    y: stat.memory,
    value: stat.memoryUsage,
  })),
);
</script>
