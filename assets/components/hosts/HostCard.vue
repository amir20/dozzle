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

      <!-- The machine's own read-outs, in one hairline chip pushed to the right
           edge and led by a pulse, not the host's own icon, which the name already
           wears. The facts on the left are about
           what Docker runs and the meters below sum the containers, so without a
           boundary "Load" and "Disk" read as more container numbers. Inside,
           labels stay muted, values carry the weight, and each hides when it is
           not known. A phone has no room beside the name; see the footer below. -->
      <div
        v-if="host.available && hasHostMetrics && !isMobile"
        class="border-base-content/10 text-base-content/50 ml-auto flex flex-wrap items-center gap-x-3 gap-y-1 rounded-md border px-2 py-0.5 text-xs tabular-nums"
        :title="$t('label.host')"
      >
        <ph:pulse class="size-3.5 opacity-60" />
        <span v-if="uptimeLabel">
          {{ $t("label.uptime") }} <span class="text-base-content/80 font-mono">{{ uptimeLabel }}</span>
        </span>
        <span v-if="host.metricsAvailable" :title="loadTitle">
          {{ $t("label.load") }} <span class="font-mono" :class="loadClass">{{ loadLabel }}</span>
        </span>
        <!-- UsageMeter's track and thresholds, inline: the component is a labelled
             block sized for a panel row, too tall for a header fact. The fill stays
             neutral below 70% so the bar only takes color when disk needs a look. -->
        <span v-if="diskPercent !== undefined" class="flex items-center gap-1.5" :title="diskTitle">
          {{ $t("label.disk") }}
          <span class="bg-base-content/10 h-1.5 w-10 overflow-hidden rounded-full">
            <span
              class="block h-full rounded-full transition-[width] duration-500"
              :class="diskPercent > 90 ? 'bg-error' : diskPercent > 70 ? 'bg-warning' : 'bg-base-content/40'"
              :style="{ width: `${Math.min(diskPercent, 100)}%` }"
            ></span>
          </span>
          <span class="text-base-content/80 font-mono">{{ diskPercent }}%</span>
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
         split into cells that span the card, each over a thin meter so the width
         it takes carries load rather than empty space. Disk is a third cell here
         rather than a chip of its own under the name. -->
    <div
      v-else-if="stats && isMobile"
      class="bg-base-content/5.5 divide-base-content/10 grid divide-x rounded-lg tabular-nums"
      :class="meters.length === 3 ? 'grid-cols-3' : 'grid-cols-2'"
    >
      <div
        v-for="meter in meters"
        :key="meter.key"
        class="flex min-w-0 flex-col gap-1.5 py-2"
        :class="meters.length === 3 ? 'px-2.5' : 'px-3'"
        :title="meter.title ?? `${meter.value} / ${meter.limit}`"
      >
        <div class="flex min-w-0 items-center gap-1.5">
          <component :is="meter.icon" class="text-base-content/40 size-3.5 shrink-0" />
          <span class="text-[13px] font-semibold">{{ meter.value }}</span>
          <!-- Three cells leave no room for the limit, and a truncated "/." read as
               a glitch; it moves to the tooltip instead. -->
          <span v-if="meters.length < 3" class="text-base-content/45 truncate text-[11px]">/ {{ meter.limit }}</span>
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

    <!-- On a phone, disk is a cell in the strip above and uptime and load trail
         it as a footer. Under the name they read as facts about Docker; led by
         the same pulse as the desktop chip and sitting under the host's disk,
         they read as the machine's. -->
    <div
      v-if="isMobile && host.available && (uptimeLabel || host.metricsAvailable)"
      class="text-base-content/50 -mt-1 flex flex-wrap items-center justify-end gap-x-2 gap-y-1 px-1 text-xs tabular-nums"
    >
      <ph:pulse class="size-3.5 opacity-60" :title="$t('label.host')" />
      <template v-if="uptimeLabel">
        <span>
          {{ $t("label.uptime") }} <span class="text-base-content/80 font-mono">{{ uptimeLabel }}</span>
        </span>
      </template>
      <template v-if="host.metricsAvailable">
        <span v-if="uptimeLabel" class="text-base-content/25">·</span>
        <span :title="loadTitle">
          {{ $t("label.load") }} <span class="font-mono" :class="loadClass">{{ loadLabel }}</span>
        </span>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from "vue";
import type { Host } from "@/stores/hosts";
import { sessionHost } from "@/composable/app/storage";
import { Container } from "@/models/Container";
import PhCpu from "~icons/ph/cpu";
import PhMemory from "~icons/ph/memory";
import PhHardDrives from "~icons/ph/hard-drives";

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

// Only the 1 minute average is shown; three bare numbers in a row meant nothing
// without a legend, so the 5 and 15 minute ones live in the tooltip.
const loadLabel = computed(() => (props.host.load1 ?? 0).toFixed(2));

const loadTitle = computed(() => {
  const { load1 = 0, load5 = 0, load15 = 0 } = props.host;
  return `1m ${load1.toFixed(2)} · 5m ${load5.toFixed(2)} · 15m ${load15.toFixed(2)} · ${t("label.core", hostCores.value)}`;
});

// A load figure only means something against the core count: 0.9 is idle on 12
// cores and saturated on 1. Past one runnable task per core it warns, past two it
// errors, the same way the disk bar takes color only when it needs a look.
const loadClass = computed(() => {
  const perCore = (props.host.load1 ?? 0) / hostCores.value;
  return perCore > 2 ? "text-error" : perCore > 1 ? "text-warning" : "text-base-content/80";
});

const uptimeLabel = computed(() => (props.host.metricsAvailable ? formatUptime(props.host.uptime) : undefined));

// Docker's own disk first, then any drive mounted under /host/disks. The bar shows
// the fullest one, since that is the one that will run out; the tooltip lists all.
const drives = computed(() => {
  const list = (props.host.disks ?? []).map(({ name, total, free }) => ({ name, total, used: total - free }));
  const total = props.host.diskTotal ?? 0;
  if (total) list.unshift({ name: runtimeLabel.value, total, used: total - (props.host.diskFree ?? 0) });
  return list.map((drive) => ({ ...drive, percent: Math.round((drive.used / drive.total) * 100) }));
});

const diskPercent = computed(() =>
  drives.value.length ? Math.max(...drives.value.map((drive) => drive.percent)) : undefined,
);

const hasHostMetrics = computed(
  () => !!uptimeLabel.value || props.host.metricsAvailable || diskPercent.value !== undefined,
);

const diskTitle = computed(() =>
  drives.value
    .map((drive) => {
      const usage = `${formatBytes(drive.used, { decimals: 1 })} / ${formatBytes(drive.total, { decimals: 1 })}`;
      return drives.value.length > 1 ? `${drive.name} ${usage} (${drive.percent}%)` : usage;
    })
    .join("\n"),
);

const stats = reactive({ mostRecent: totalStat, weighted: useExponentialMovingAverage(totalStat) });

// The fullest drive, since that is the one the bar shows.
const fullestDrive = computed(() =>
  drives.value.length ? drives.value.reduce((a, b) => (b.percent > a.percent ? b : a)) : undefined,
);

const meters = computed(() => {
  const { totalCPU, totalMemUsage } = stats.weighted.movingAverage;
  const disk = fullestDrive.value;
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
    ...(disk
      ? [
          {
            key: "disk",
            icon: PhHardDrives,
            value: `${disk.percent}%`,
            limit: formatBytes(disk.total, { short: true, decimals: 0 }),
            percent: disk.percent,
            bar: "bg-accent",
            title: diskTitle.value,
          },
        ]
      : []),
  ] as { key: string; icon: Component; value: string; limit: string; percent: number; bar: string; title?: string }[];
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
