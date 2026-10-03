<template>
  <!-- One surface, hairline-separated. Three separately tinted cards made the
       toolbar read as three competing things; here the sparklines are the only
       color, so the eye lands on the numbers. -->
  <div class="bg-base-content/[0.055] divide-base-content/10 flex items-stretch divide-x rounded-lg">
    <IOCard
      :network-rx="networkRate.rx"
      :network-tx="networkRate.tx"
      :disk-read="diskRate.read"
      :disk-write="diskRate.write"
    />

    <StatCard
      label="CPU"
      class="md:min-w-52"
      :title="t('tooltip.cpu-usage', { cpu: totalStat.cpu.toFixed(2), cores: roundCPU(limits.cpu) })"
    >
      <template #icon><ph:cpu class="size-3.5" /></template>
      <template #value="{ hoveredValue }">
        <span class="tabular-nums">
          <span class="text-[13px] font-semibold"> {{ Math.max(0, hoveredValue ?? totalStat.cpu).toFixed(1) }}% </span>
          <span class="text-base-content/45 text-[11px] max-md:hidden"> / {{ roundCPU(limits.cpu) }}</span>
        </span>
      </template>
      <template #chart="{ onHoverValue, onHoverEnd }">
        <BarChart
          ref="cpuChart"
          :chart-data="cpuData"
          :sampled-from="sampledFrom"
          bar-class="text-primary"
          class="h-4 w-full max-md:hidden"
          @hover-value="onHoverValue"
          @hover-end="onHoverEnd"
        />
      </template>
    </StatCard>

    <StatCard
      label="MEM"
      class="md:min-w-52"
      :title="
        t('tooltip.memory-usage', { used: formatBytes(totalStat.memoryUsage), total: formatBytes(limits.memory) })
      "
    >
      <template #icon><ph:memory class="size-3.5" /></template>
      <template #value="{ hoveredValue }">
        <span class="tabular-nums">
          <span class="text-[13px] font-semibold">{{
            formatBytes(hoveredValue ?? totalStat.memoryUsage, { short: true, decimals: 1 })
          }}</span>
          <span class="text-base-content/45 text-[11px] max-md:hidden">
            / {{ formatBytes(limits.memory, { short: true, decimals: 1 }) }}</span
          >
        </span>
      </template>
      <template #chart="{ onHoverValue, onHoverEnd }">
        <BarChart
          ref="memoryChart"
          :chart-data="memoryData"
          :sampled-from="sampledFrom"
          bar-class="text-secondary"
          class="h-4 w-full max-md:hidden"
          @hover-value="onHoverValue"
          @hover-end="onHoverEnd"
        />
      </template>
    </StatCard>
  </div>
</template>

<script lang="ts" setup>
import { Container, Stat, emptyStat, isStopped } from "@/models/Container";
import StatCard from "@/components/ui/StatCard.vue";
import IOCard from "@/components/ui/IOCard.vue";
import BarChart from "@/components/ui/BarChart.vue";

const { containers } = defineProps<{
  containers: Container[];
}>();

const { t } = useI18n();

// shallow, for the reason given on HostCard's own totalStat.
const totalStat = shallowRef<Stat>(emptyStat());
const { history, reset } = useSimpleRefHistory(totalStat, { capacity: 300 });

// The padded head of the series, which is not data. See `Container.statsHistory`.
const sampledCount = ref(0);
const sampledFrom = computed(() => Math.max(0, history.value.length - sampledCount.value));
const { hosts } = useHosts();
const cpuChart = useTemplateRef("cpuChart");
const memoryChart = useTemplateRef("memoryChart");
const networkRate = ref({ rx: 0, tx: 0 });
const diskRate = ref({ read: 0, write: 0 });

const roundCPU = (num: number) => (Number.isInteger(num) ? num.toFixed(0) : num.toFixed(1));

// What the total is measured against. A stopped container is not using its share,
// so it only counts when nothing in the view is running.
const limits = computed(() => {
  const running = containers.filter((c) => !isStopped(c));
  const counted = running.length > 0 ? running : containers;
  const containersByHost = new Map<string, Container[]>();
  counted.forEach((container) => {
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

// Docker reports cpu as 100 per core, so the raw figures add up and the sum is then
// taken against the cores the view can use. Adding each container's own percentage
// instead mixed denominators: two half-core containers at full load read 200% of one.
// A stopped container keeps its last sample, which is not what it uses now, so it adds
// no cpu or memory. Its I/O counters stay in so the totals do not drop and fake a rate.
function sum(samples: { container: Container; stat: Stat }[]): Stat {
  const raw = samples.reduce((acc, { container, stat }) => {
    const running = !isStopped(container);
    return {
      cpu: acc.cpu + (running ? stat.cpu : 0),
      memory: acc.memory + (running ? stat.memory : 0),
      memoryUsage: acc.memoryUsage + (running ? stat.memoryUsage : 0),
      networkRxTotal: acc.networkRxTotal + stat.networkRxTotal,
      networkTxTotal: acc.networkTxTotal + stat.networkTxTotal,
      diskReadTotal: acc.diskReadTotal + stat.diskReadTotal,
      diskWriteTotal: acc.diskWriteTotal + stat.diskWriteTotal,
    };
  }, emptyStat());
  const { cpu, memory } = limits.value;
  return {
    ...raw,
    cpu: raw.cpu / (cpu || 1),
    memory: memory > 0 ? (raw.memoryUsage / memory) * 100 : raw.memory,
  };
}

// Keyed on the ids rather than the array, which the store hands over fresh on every
// list update: reseeding then would paint a stopped container's history back in.
watch(
  () => containers.map((c) => c.id).join(","),
  () => {
    const initial: Stat[] = [];
    for (let i = 1; i <= 300; i++) {
      initial.push(
        sum(
          containers.flatMap((container) => {
            const stat = container.statsHistory.at(-i);
            return stat ? [{ container, stat }] : [];
          }),
        ),
      );
    }
    totalStat.value = initial[0];
    reset({ initial: initial.reverse() });
    // `max`, not `min`: the two only differ when one container's history is
    // shorter than another's, which means that container did not exist yet, and
    // zero is its honest contribution to a total. Taking the min would let one
    // newly created container blank the sampled region for everything else.
    sampledCount.value = Math.min(300, Math.max(0, ...containers.map((c) => c.sampledStats)));
    // Charts cache their downsampled bars and only patch the last bar per tick;
    // a container switch replaces the whole series, so force a full recalculate.
    nextTick(() => {
      cpuChart.value?.recalculate();
      memoryChart.value?.recalculate();
    });
  },
  { immediate: true },
);

useIntervalFn(() => {
  const previousStat = totalStat.value;
  totalStat.value = sum(containers.map((container) => ({ container, stat: container.stat })));
  sampledCount.value = Math.min(300, sampledCount.value + 1);

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
