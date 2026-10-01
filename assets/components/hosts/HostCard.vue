<template>
  <div
    class="border-base-content/10 bg-base-100 hover:border-base-content/20 rounded-box flex flex-col gap-3 border p-4 transition-colors"
  >
    <!-- Name and facts share one line: the facts are short, and a second line for
         them made the header as tall as the meters it introduces. They wrap under
         the name only when the card is too narrow for both. -->
    <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
      <div class="flex min-w-0 items-center gap-2">
        <!-- Connection state rides on the host's own icon as a dot, the way a
             container's does, so a healthy header carries no color at all. -->
        <span class="bg-base-content/5 text-base-content/70 relative flex-none rounded-md p-1.5" :title="agentTooltip">
          <HostIcon :type="host.type" class="size-4" />
          <span
            class="ring-base-100 absolute -top-0.5 -right-0.5 size-2 rounded-full ring-2"
            :class="host.available ? 'bg-success' : 'bg-base-content/30'"
          ></span>
        </span>
        <!-- Selects the host in the sidebar, the same as clicking its row there.
             Not a link to /host/[id]: that is the merged stream, which is what the
             sidebar's merge button is for. -->
        <button
          type="button"
          class="hover:decoration-base-content/40 truncate text-left text-lg font-semibold tracking-tight hover:underline hover:underline-offset-4"
          :title="host.name"
          @click="sessionHost = host.id"
        >
          {{ host.name }}
        </button>
      </div>

      <!-- Plain facts, no glyph per fact. The agent version only appears when it
           disagrees with the server: a matching one said nothing and read like a
           second runtime version. -->
      <div class="text-base-content/50 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs tabular-nums">
        <template v-if="host.available">
          <span>{{ $t("label.container", hostContainers.length) }}</span>
          <span class="text-base-content/25">·</span>
          <span>{{ runtimeLabel }} {{ host.dockerVersion }}</span>
        </template>
        <span v-else>{{ $t("label.host-unreachable") }}</span>
        <span
          v-if="agentOutdated"
          class="status-pill status-pill-warning ml-1"
          :title="$t('tooltip.agent-version-mismatch', { version: host.agentVersion, current: config.version })"
        >
          {{ $t("label.agent-outdated", { version: host.agentVersion }) }}
        </span>
      </div>
    </div>

    <!-- An offline host has no live numbers, so the meters give way to one line in
         the same slot rather than showing the last values as if they were current. -->
    <div
      v-if="!host.available"
      class="bg-base-content/5.5 text-base-content/50 flex items-center gap-2 rounded-lg px-3 py-2.5 text-xs"
    >
      <mdi:lan-disconnect class="size-3.5 shrink-0 opacity-60" />
      <i18n-t keypath="label.agent-unreachable" tag="span" class="truncate">
        <template #endpoint>
          <span class="font-mono">{{ host.endpoint }}</span>
        </template>
      </i18n-t>
    </div>

    <!-- Two charts at half a phone's width are too narrow to read a trend from and
         push the container list below the fold, so a phone gets the numbers alone,
         split into two halves that span the card, each over a thin meter so the
         width it takes carries load rather than empty space. -->
    <div
      v-else-if="stats && isMobile"
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
        ref="cpuCard"
        :icon="PhCpu"
        label="CPU"
        :capacity="$t('label.core', host.nCPU ?? 0)"
        :value="stats.weighted.movingAverage.totalCPU"
        :chartData="cpuHistory"
        :sample-interval="SAMPLE_INTERVAL"
        :sampled-from="sampledFrom"
        text-class="text-primary"
        bar-class="text-primary"
        :formatValue="(value) => `${value.toFixed(1)}%`"
      />

      <MetricCard
        ref="memCard"
        :icon="PhMemory"
        label="Memory"
        :capacity="formatBytes(host.memTotal, { decimals: 1 })"
        :value="stats.weighted.movingAverage.totalMemUsage"
        :chartData="memHistory"
        :sample-interval="SAMPLE_INTERVAL"
        :sampled-from="sampledFrom"
        :chart-max="100"
        text-class="text-secondary"
        bar-class="text-secondary"
        :formatValue="(value) => formatBytes(value, { decimals: 1 })"
      />
    </div>

    <!-- Host facts (disk, load, uptime, network) read from the host /proc. -->
    <div
      v-if="host.available && hostFacts.length"
      class="border-base-content/10 text-base-content/60 flex flex-wrap items-center gap-x-4 gap-y-1 border-t pt-2 text-xs tabular-nums"
    >
      <span v-for="fact in hostFacts" :key="fact.label" class="flex items-center gap-1.5">
        <span class="text-base-content/40">{{ fact.label }}</span>
        <span class="text-base-content/80 font-medium">{{ fact.value }}</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Host } from "@/stores/hosts";
import { sessionHost } from "@/composable/app/storage";
import { Container } from "@/models/Container";
import PhCpu from "~icons/ph/cpu";
import PhMemory from "~icons/ph/memory";

const props = defineProps<{
  host: Host;
}>();

// How often the totals below are sampled, and therefore how far apart two points
// of the history are. The backfill from `statsHistory` assumes the same cadence,
// which is what lets a hovered bar name a time.
const SAMPLE_INTERVAL = 1000;

const { t } = useI18n();
const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
};

const hostContainers = computed(() =>
  containers.value.filter((container) => container.host === props.host.id && container.state === "running"),
);

const runtimeLabel = computed(() => (props.host.runtime === "podman" ? "Podman" : "Docker"));

const agentTooltip = computed(() =>
  props.host.type === "agent" && props.host.agentVersion
    ? t("tooltip.agent-version", { version: props.host.agentVersion })
    : undefined,
);

const agentOutdated = computed(() => props.host.type === "agent" && props.host.agentVersion !== config.version);

// Every container is divided by the host's cores, limited or not. Dividing by
// `cpuLimit` reads as how close a container is to being throttled, which is right
// on its own row but adds up to nonsense here: the card is a share of the host.
const hostCores = computed(() => props.host.nCPU || 1);

type TotalStat = {
  totalCPU: number;
  totalMem: number;
  totalMemUsage: number;
};

// shallow: replaced wholesale each tick and never edited in place, so the deep
// proxy a plain `ref` would build over it is pure cost. See useSimpleRefHistory.
const totalStat = shallowRef<TotalStat>({ totalCPU: 0, totalMem: 0, totalMemUsage: 0 });
const { history, reset } = useSimpleRefHistory(totalStat, { capacity: 300 });

// How many entries at the end of `history` are real totals. The backfill below
// seeds it from the containers' own sample counts and each tick adds one, so the
// padded head shrinks as the series scrolls in.
const sampledCount = ref(0);
const sampledFrom = computed(() => Math.max(0, history.value.length - sampledCount.value));

const cpuCard = useTemplateRef("cpuCard");
const memCard = useTemplateRef("memCard");

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

const formatUptime = (secs?: number) => {
  if (!secs) return undefined;
  const d = Math.floor(secs / 86400);
  const h = Math.floor((secs % 86400) / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
};

const diskPercent = computed(() => {
  const total = props.host.diskTotal ?? 0;
  const free = props.host.diskFree ?? 0;
  return total > 0 ? ((total - free) / total) * 100 : 0;
});

const hostFacts = computed(() => {
  const facts: { label: string; value: string }[] = [];
  if (props.host.diskTotal) {
    const used = props.host.diskTotal - (props.host.diskFree ?? 0);
    facts.push({
      label: "Disk",
      value: `${formatBytes(used, { short: true, decimals: 1 })} / ${formatBytes(props.host.diskTotal, { short: true, decimals: 1 })} (${diskPercent.value.toFixed(0)}%)`,
    });
  }
  if (props.host.load1 !== undefined) {
    const l = [props.host.load1, props.host.load5, props.host.load15].filter((v) => v !== undefined && v !== null);
    facts.push({ label: "Load", value: l.map((v) => (v as number).toFixed(2)).join(" ") });
  }
  const up = formatUptime(props.host.uptime);
  if (up) facts.push({ label: "Uptime", value: up });
  if (props.host.netRxTotal !== undefined) {
    facts.push({
      label: "Net",
      value: `↓ ${formatBytes(props.host.netRxTotal ?? 0, { short: true, decimals: 1 })}  ↑ ${formatBytes(props.host.netTxTotal ?? 0, { short: true, decimals: 1 })}`,
    });
  }
  return facts;
});

const stats = reactive({ mostRecent: totalStat, weighted: useExponentialMovingAverage(totalStat) });

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
          return {
            totalCPU: acc.totalCPU + item.cpu / hostCores.value,
            totalMem: acc.totalMem + item.memory,
            totalMemUsage: acc.totalMemUsage + item.memoryUsage,
          };
        },
        { totalCPU: 0, totalMem: 0, totalMemUsage: 0 },
      );
      initial.push(stat);
    }
    reset({ initial: initial.reverse() });
    // `max`, not `min`: the two only differ when one container's history is
    // shorter than another's, which means that container did not exist yet, and
    // zero is its honest contribution to a total. Taking the min would let one
    // newly created container blank the sampled region for everything else.
    sampledCount.value = Math.min(300, Math.max(0, ...hostContainers.value.map((c) => c.sampledStats)));
    stats.weighted.reset(initial.at(-1)!);
    // The backfill replaces the series outright, but its length is still 300, so
    // nothing in the chart notices: it would keep the old bars, and the old sampled
    // flags with them, until the next scheduled recalculation. Hovering in between
    // reported padding as a measurement, which is the thing these flags exist to
    // prevent. MultiContainerStat does the same for the same reason.
    nextTick(() => {
      cpuCard.value?.recalculate();
      memCard.value?.recalculate();
    });
  },
  { immediate: true },
);

useIntervalFn(() => {
  totalStat.value = hostContainers.value.reduce(
    (acc, container) => {
      return {
        totalCPU: acc.totalCPU + container.stat.cpu / hostCores.value,
        totalMem: acc.totalMem + container.stat.memory,
        totalMemUsage: acc.totalMemUsage + container.stat.memoryUsage,
      };
    },
    { totalCPU: 0, totalMem: 0, totalMemUsage: 0 },
  );
  sampledCount.value = Math.min(300, sampledCount.value + 1);
}, SAMPLE_INTERVAL);
</script>
