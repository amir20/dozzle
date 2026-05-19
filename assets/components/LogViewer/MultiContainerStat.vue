<template>
  <div class="flex items-stretch gap-2.5">
    <IOCard
      :network-rx="networkRate.rx"
      :network-tx="networkRate.tx"
      :disk-read="diskRate.read"
      :disk-write="diskRate.write"
    />

    <StatCard
      :icon="PhCpu"
      card-class="bg-primary/10 md:min-w-56"
      icon-class="text-primary"
      :title="`CPU ${totalStat.cpu.toFixed(2)}% / ${roundCPU(limits.cpu)} cores`"
    >
      <template #value="{ hoveredValue }">
        <span class="tabular-nums">
          <span class="font-semibold"> {{ Math.max(0, hoveredValue ?? totalStat.cpu).toFixed(2) }}% </span>
          <span class="text-base-content/60 max-md:hidden"> / {{ roundCPU(limits.cpu) }} CPU</span>
        </span>
      </template>
      <template #chart="{ onHoverValue }">
        <Sparkline :data="cpuData" bar-class="bg-primary" class="max-md:hidden" @hover-value="onHoverValue" />
      </template>
    </StatCard>

    <StatCard
      :icon="PhMemory"
      card-class="bg-secondary/10 md:min-w-56"
      icon-class="text-secondary"
      :title="`Memory ${formatBytes(totalStat.memoryUsage)} / ${formatBytes(limits.memory)}`"
    >
      <template #value="{ hoveredValue }">
        <span class="tabular-nums">
          <span class="font-semibold">{{ formatBytes(hoveredValue ?? totalStat.memoryUsage) }}</span>
          <span class="text-base-content/60 max-md:hidden">
            / {{ formatBytes(limits.memory, { short: true, decimals: 1 }) }}</span
          >
        </span>
      </template>
      <template #chart="{ onHoverValue }">
        <Sparkline :data="memoryData" bar-class="bg-secondary" class="max-md:hidden" @hover-value="onHoverValue" />
      </template>
    </StatCard>
  </div>
</template>

<script lang="ts" setup>
import { Stat } from "@/models/Container";
import { Container } from "@/models/Container";
import StatCard from "@/components/LogViewer/StatCard.vue";
import IOCard from "@/components/LogViewer/IOCard.vue";
import Sparkline from "@/components/LogViewer/Sparkline.vue";
// @ts-ignore
import PhCpu from "~icons/ph/cpu";
// @ts-ignore
import PhMemory from "~icons/ph/memory";

const { containers } = defineProps<{
  containers: Container[];
}>();

const totalStat = ref<Stat>({
  cpu: 0,
  memory: 0,
  memoryUsage: 0,
  networkRxTotal: 0,
  networkTxTotal: 0,
  diskReadTotal: 0,
  diskWriteTotal: 0,
});
const { history, reset } = useSimpleRefHistory(totalStat, { capacity: 300 });
const { hosts } = useHosts();
const networkRate = ref({ rx: 0, tx: 0 });
const diskRate = ref({ read: 0, write: 0 });

const roundCPU = (num: number) => (Number.isInteger(num) ? num.toFixed(0) : num.toFixed(1));

function toContainerCores(container: Container): number {
  if (container.cpuLimit && container.cpuLimit > 0) {
    return container.cpuLimit;
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
            diskReadTotal: acc.diskReadTotal + item.diskReadTotal,
            diskWriteTotal: acc.diskWriteTotal + item.diskWriteTotal,
          };
        },
        {
          cpu: 0,
          memory: 0,
          memoryUsage: 0,
          networkRxTotal: 0,
          networkTxTotal: 0,
          diskReadTotal: 0,
          diskWriteTotal: 0,
        },
      );
      initial.push(stat);
    }
    totalStat.value = initial[0];
    reset({ initial: initial.reverse() });
  },
  { immediate: true },
);

const limits = computed(() => {
  const containersByHost = new Map<string, Container[]>();
  containers.forEach((container) => {
    if (!containersByHost.has(container.host)) {
      containersByHost.set(container.host, []);
    }
    containersByHost.get(container.host)!.push(container);
  });

  let totalCpu = 0;
  let totalMemory = 0;

  containersByHost.forEach((hostContainers, hostId) => {
    const hostInfo = hosts.value[hostId];
    const hostTotalMemory = hostInfo?.memTotal || 0;
    const hostTotalCpu = hostInfo?.nCPU || 0;

    const hasUnlimitedCpu = hostContainers.some((c) => !c.cpuLimit || c.cpuLimit <= 0);
    const hasUnlimitedMemory = hostContainers.some((c) => !c.memoryLimit);

    if (hasUnlimitedCpu) {
      totalCpu += hostTotalCpu;
    } else {
      const sumCpu = hostContainers.reduce((sum, c) => sum + (c.cpuLimit || 0), 0);
      totalCpu += Math.min(sumCpu, hostTotalCpu);
    }

    if (hasUnlimitedMemory) {
      totalMemory += hostTotalMemory;
    } else {
      const sumMemory = hostContainers.reduce((sum, c) => sum + (c.memoryLimit || 0), 0);
      totalMemory += Math.min(sumMemory, hostTotalMemory);
    }
  });

  return { cpu: totalCpu, memory: totalMemory };
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
        diskReadTotal: acc.diskReadTotal + container.stat.diskReadTotal,
        diskWriteTotal: acc.diskWriteTotal + container.stat.diskWriteTotal,
      };
    },
    { cpu: 0, memory: 0, memoryUsage: 0, networkRxTotal: 0, networkTxTotal: 0, diskReadTotal: 0, diskWriteTotal: 0 },
  );

  networkRate.value = {
    rx: Math.max(0, totalStat.value.networkRxTotal - previousStat.networkRxTotal),
    tx: Math.max(0, totalStat.value.networkTxTotal - previousStat.networkTxTotal),
  };
  diskRate.value = {
    read: Math.max(0, totalStat.value.diskReadTotal - previousStat.diskReadTotal),
    write: Math.max(0, totalStat.value.diskWriteTotal - previousStat.diskWriteTotal),
  };
}, 1000);

const cpuData = computed(() =>
  history.value.map((stat) => ({
    percent: Math.max(0, stat.cpu),
    value: Math.max(0, stat.cpu),
  })),
);

const memoryData = computed(() =>
  history.value.map((stat) => ({
    percent: stat.memory,
    value: stat.memoryUsage,
  })),
);
</script>
