<template>
  <div
    class="border-base-content/10 bg-base-100 hover:border-base-content/20 rounded-box flex flex-col gap-3 border p-4 transition-colors"
  >
    <!-- Name and facts share one line: the facts are short, and a second line for
         them made the header as tall as the meters it introduces. They wrap under
         the name only when the card is too narrow for both. -->
    <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
      <div class="flex min-w-0 items-center gap-2">
        <span class="bg-base-content/5 text-base-content/70 flex-none rounded-md p-1.5">
          <HostIcon :type="host.type" class="size-4" />
        </span>
        <div class="truncate text-lg font-semibold tracking-tight" :title="host.name">{{ host.name }}</div>

        <span class="status-pill status-pill-error flex-none gap-1" v-if="!host.available">
          <carbon:warning class="size-3" />
          offline
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
        <li
          class="flex items-center gap-1.5"
          :class="{ 'text-warning': agentOutdated }"
          v-if="host.type == 'agent' && host.agentVersion"
          :title="
            agentOutdated
              ? $t('tooltip.agent-version-mismatch', { version: host.agentVersion, current: config.version })
              : $t('tooltip.agent-version', { version: host.agentVersion })
          "
        >
          <carbon:warning v-if="agentOutdated" class="size-3.5" />
          <mdi:satellite-variant v-else class="size-3.5" />
          {{ host.agentVersion }}
        </li>
      </ul>
    </div>

    <!-- Two charts at half a phone's width are too narrow to read a trend from and
         push the container list below the fold, so a phone gets the numbers alone,
         split into two halves that span the card, each over a thin meter so the
         width it takes carries load rather than empty space. -->
    <div
      v-if="stats && isMobile"
      class="bg-base-content/5.5 divide-base-content/10 grid grid-cols-2 divide-x rounded-lg tabular-nums"
    >
      <div v-for="meter in meters" :key="meter.key" class="flex min-w-0 flex-col gap-1.5 px-3 py-2">
        <div class="flex min-w-0 items-center gap-1.5">
          <component :is="meter.icon" class="text-base-content/40 size-3.5 shrink-0" />
          <span class="text-[13px] font-semibold">{{ meter.value }}</span>
          <span class="text-base-content/45 truncate text-[11px]">/ {{ meter.limit }}</span>
        </div>
        <div class="bg-base-content/10 h-1 overflow-hidden rounded-full">
          <div
            class="h-full rounded-full transition-[width] duration-500"
            :class="meter.percent > 90 ? 'bg-error' : meter.percent > 70 ? 'bg-warning' : meter.bar"
            :style="{ width: `${Math.min(Math.max(meter.percent, 0), 100)}%` }"
          ></div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3" v-else-if="stats">
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
        :chart-max="100"
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

const agentOutdated = computed(() => props.host.type === "agent" && props.host.agentVersion !== config.version);

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
    // Against the host's memory, so the bars read as how full it is.
    percent: props.host.memTotal ? (stat.totalMemUsage / props.host.memTotal) * 100 : 0,
    value: stat.totalMemUsage,
  })),
);

const stats = reactive({ mostRecent: totalStat, weighted: useExponentialMovingAverage(totalStat) });

const { t } = useI18n();
const meters = computed(() => {
  const { totalCPU, totalMemUsage } = stats.weighted.movingAverage;
  return [
    {
      key: "cpu",
      icon: PhCpu,
      value: `${totalCPU.toFixed(1)}%`,
      limit: t("label.core", props.host.nCPU ?? 0),
      percent: totalCPU,
      bar: "bg-primary",
    },
    {
      key: "mem",
      icon: PhMemory,
      value: formatBytes(totalMemUsage, { short: true, decimals: 1 }),
      limit: formatBytes(props.host.memTotal, { short: true, decimals: 1 }),
      percent: props.host.memTotal ? (totalMemUsage / props.host.memTotal) * 100 : 0,
      bar: "bg-secondary",
    },
  ];
});

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
