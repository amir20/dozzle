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
      :title="
        t('tooltip.cpu-usage', {
          cpu: cpuDisplayValue(totalStat.cpu, rawCpuTotal).toFixed(2),
          cores: roundCPU(limits.cpu),
        })
      "
    >
      <template #value="{ hoveredValue }">
        <span class="tabular-nums">
          <span class="font-semibold">
            {{ cpuDisplayValue(hoveredValue ?? totalStat.cpu, (hoveredValue ?? totalStat.cpu) * cpuScale).toFixed(1) }}%
          </span>
          <span class="text-base-content/60 max-md:hidden"> / {{ roundCPU(limits.cpu) }} CPU</span>
        </span>
      </template>
      <template #chart="{ onHoverValue }">
        <BarChart
          ref="cpuChart"
          :chart-data="cpuData"
          bar-class="bg-primary opacity-80 hover:opacity-100"
          class="h-5 w-full max-md:hidden"
          @hover-value="onHoverValue"
        />
      </template>
    </StatCard>

    <StatCard
      :icon="PhMemory"
      card-class="bg-secondary/10 md:min-w-56"
      icon-class="text-secondary"
      :title="
        t('tooltip.memory-usage', { used: formatBytes(totalStat.memoryUsage), total: formatBytes(limits.memory) })
      "
    >
      <template #value="{ hoveredValue }">
        <span class="tabular-nums">
          <span class="font-semibold">{{
            formatBytes(hoveredValue ?? totalStat.memoryUsage, { short: true, decimals: 1 })
          }}</span>
          <span class="text-base-content/60 max-md:hidden">
            / {{ formatBytes(limits.memory, { short: true, decimals: 1 }) }}</span
          >
        </span>
      </template>
      <template #chart="{ onHoverValue }">
        <BarChart
          ref="memoryChart"
          :chart-data="memoryData"
          bar-class="bg-secondary opacity-80 hover:opacity-100"
          class="h-5 w-full max-md:hidden"
          @hover-value="onHoverValue"
        />
      </template>
    </StatCard>
  </div>
</template>

<script lang="ts" setup>
import { Container, Stat, emptyStat } from "@/models/Container";
import StatCard from "@/components/LogViewer/StatCard.vue";
import IOCard from "@/components/LogViewer/IOCard.vue";
import BarChart from "@/components/BarChart.vue";
import PhCpu from "~icons/ph/cpu";
import PhMemory from "~icons/ph/memory";

const { containers } = defineProps<{
  containers: Container[];
}>();

const { t } = useI18n();

const totalStat = ref<Stat>(emptyStat());
const { history, reset } = useSimpleRefHistory(totalStat, { capacity: 300 });
const { hosts } = useHosts();
const cpuChart = useTemplateRef("cpuChart");
const memoryChart = useTemplateRef("memoryChart");
const networkRate = ref({ rx: 0, tx: 0 });
const diskRate = ref({ read: 0, write: 0 });

const roundCPU = (num: number) => (Number.isInteger(num) ? num.toFixed(0) : num.toFixed(1));

// Raw per-core CPU total (100 == one core) used for the "cores" (Linux style) display.
const rawCpuTotal = computed(() =>
  Math.max(
    0,
    containers.reduce((acc, container) => acc + container.stat.cpu, 0),
  ),
);
// Ratio between the per-core total and the whole-CPU utilization total, so historical
// (hovered) utilization values can be rescaled to the per-core form.
const cpuScale = computed(() => (totalStat.value.cpu > 0 ? rawCpuTotal.value / totalStat.value.cpu : 1));

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
      const stat = containers.reduce((acc, container) => {
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
      }, emptyStat());
      initial.push(stat);
    }
    totalStat.value = initial[0];
    reset({ initial: initial.reverse() });
    // Charts cache their downsampled bars and only patch the last bar per tick;
    // a container switch replaces the whole series, so force a full recalculate.
    nextTick(() => {
      cpuChart.value?.recalculate();
      memoryChart.value?.recalculate();
    });
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
  totalStat.value = containers.reduce((acc, container) => {
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
  }, emptyStat());

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
