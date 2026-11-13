<template>
  <!--
    What happened under the charts above it: a deploy as a ring, an alert as a
    solid dot in its severity, each placed on the same time axis the bars use.
    A step in CPU or memory is usually one of these, and without them the chart
    shows that something changed but never what.

    Alerts are the same AlertDot the container table uses, so one alert looks
    and behaves like itself everywhere, "Show me the lines" included.
  -->
  <div v-if="marks.length" class="relative h-3" data-timeline>
    <div class="bg-base-content/10 absolute inset-x-0 top-1/2 h-px"></div>
    <div
      v-for="mark in marks"
      :key="mark.key"
      class="absolute top-1/2 flex -translate-1/2"
      :style="{ left: `${mark.left}%` }"
      :data-mark="mark.kind"
    >
      <AlertDot v-if="mark.kind === 'alert'" :alert="mark.alert" />
      <Popover
        v-else
        hover
        placement="bottom-end"
        panel-class="rounded-box bg-base-200 border-base-content/10 border shadow-lg"
      >
        <template #trigger>
          <button type="button" class="deploy-mark text-info" :aria-label="deployLabel(mark.deploy)"></button>
        </template>
        <div class="flex flex-col gap-1 p-3 text-sm">
          <span class="font-semibold">{{ $t(`update-marker.${updateKind(mark.deploy)}`) }}</span>
          <span class="font-mono text-xs wrap-anywhere">
            <span class="opacity-60">{{ updateLabels(mark.deploy).from }}</span>
            <span class="mx-1 opacity-40">&rarr;</span>
            <span>{{ updateLabels(mark.deploy).to }}</span>
          </span>
          <span class="text-base-content/60 text-xs">{{ format(mark.ts) }}</span>
        </div>
      </Popover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { CloudAlert } from "@/composable/cloud/cloudAlerts";
import type { RecentDeploy } from "@/composable/cloud/recentDeploys";
import { updateKind, updateLabels } from "@/models/ContainerUpdate";

const { from, to, alerts, deploys, format } = defineProps<{
  // The window the bars above cover, in ms. A mark outside it is not drawn.
  from: number;
  to: number;
  alerts: CloudAlert[];
  deploys: RecentDeploy[];
  format: (ms: number) => string;
}>();

const { t } = useI18n();

type Mark =
  | { kind: "alert"; key: string; ts: number; left: number; alert: CloudAlert }
  | { kind: "deploy"; key: string; ts: number; left: number; deploy: RecentDeploy };

const marks = computed<Mark[]>(() => {
  const span = to - from;
  if (span <= 0) return [];
  const left = (ts: number) => ((ts - from) / span) * 100;
  const inWindow = (ts: number) => ts >= from && ts <= to;

  const result: Mark[] = [];
  for (const deploy of deploys) {
    const ts = Date.parse(deploy.at);
    if (inWindow(ts))
      result.push({ kind: "deploy", key: `d${deploy.oldId}/${deploy.newId}`, ts, left: left(ts), deploy });
  }
  // Alerts carry nanoseconds.
  for (const alert of alerts) {
    const ts = alert.ts / 1e6;
    if (inWindow(ts)) result.push({ kind: "alert", key: `a${alert.alertId}`, ts, left: left(ts), alert });
  }
  // Oldest first, so a later mark draws over an earlier one it sits on.
  return result.sort((a, b) => a.ts - b.ts);
});

const deployLabel = (deploy: RecentDeploy) => `${t(`update-marker.${updateKind(deploy)}`)} ${updateLabels(deploy).to}`;
</script>

<style scoped>
@reference "@/main.css";

/* A ring rather than a dot, as on ContainerUpdateDot, so a deploy never reads
   as an alert. The same 8px ring as ContainerUpdateDot, with the invisible hit
   area AlertDot uses. */
.deploy-mark {
  @apply bg-base-100 relative size-2 shrink-0 rounded-full border-[1.5px] border-current transition-[box-shadow,transform];
}

.deploy-mark::after {
  content: "";
  @apply absolute -inset-2;
}

.deploy-mark:hover,
.deploy-mark:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in oklab, currentColor 30%, transparent);
}

@media (prefers-reduced-motion: no-preference) {
  .deploy-mark:hover {
    transform: scale(1.2);
  }
}
</style>
