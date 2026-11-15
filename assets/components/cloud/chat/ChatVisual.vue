<template>
  <!--
    One picture under an answer. Cloud measured every number and the assistant
    only chose which picture to show, so this draws what came and never
    recomputes it. Each type is built from the component that already draws the
    same thing elsewhere in Dozzle: the metrics panel's bars, the search
    volume strip, the log viewer's level dot and timestamp.
  -->
  <div class="border-base-content/10 rounded-lg border p-3 text-xs" :data-visual="visual.type">
    <div class="mb-2 flex items-baseline gap-2">
      <span class="truncate font-mono font-semibold">{{ title }}</span>
      <span v-if="visual.hostName" class="text-base-content/50 truncate">{{ visual.hostName }}</span>
      <span v-if="aside" class="ml-auto shrink-0 font-mono tabular-nums">{{ aside }}</span>
    </div>

    <template v-if="visual.type === 'memory_vs_limit'">
      <p v-if="facts" class="mb-2">{{ facts }}</p>
      <p v-if="visual.facts?.capped" class="text-base-content/50 mb-2">{{ $t("cloud-chat.visual.floor") }}</p>
      <div v-if="bars.length">
        <span class="text-base-content/60 font-semibold tracking-wide uppercase">{{
          $t("cloud-chat.visual.memory")
        }}</span>
        <BarChart :chart-data="bars" bar-class="text-secondary" :max="limit ? 100 : undefined" class="mt-1 h-16" />
      </div>
    </template>

    <SearchVolume
      v-else-if="visual.type === 'pattern_count' && visual.volume?.length"
      :buckets="visual.volume"
      :step-ns="visual.stepNs ?? 0"
      :end="visual.to"
    />

    <ul v-else-if="visual.type === 'container_ranking'" class="space-y-1.5">
      <li v-for="row in visual.rows ?? []" :key="`${row.hostId}/${row.container}`">
        <div class="flex items-baseline gap-2">
          <span class="truncate font-mono">{{ row.container }}</span>
          <span v-if="row.hostName" class="text-base-content/50 truncate">{{ row.hostName }}</span>
          <span class="ml-auto shrink-0 font-mono tabular-nums">{{ rankValue(row.value) }}</span>
        </div>
        <div class="bg-base-content/10 mt-0.5 h-1 rounded-full">
          <div class="bg-primary h-1 rounded-full" :style="{ width: `${(row.value / rankMax) * 100}%` }"></div>
        </div>
      </li>
    </ul>

    <ul v-else-if="visual.type === 'logs'" class="space-y-1 font-mono">
      <li v-for="(line, i) in visual.lines ?? []" :key="i" class="flex items-start gap-2">
        <LogLevel :level="line.level as Level" />
        <LogDate v-if="line.timestamp" :date="new Date(line.timestamp)" />
        <span class="min-w-0 truncate" :title="line.message">{{ line.message }}</span>
      </li>
    </ul>

    <template v-else-if="visual.type === 'container_events'">
      <p>{{ eventsLine }}</p>
      <p v-if="visual.facts?.capped" class="text-base-content/50 mt-2">{{ $t("cloud-chat.visual.floor") }}</p>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { Level } from "@/models/LogEntry";
import { factsLine, memoryBars, type ChatVisual } from "@/composable/cloud/chatVisuals";

const { visual } = defineProps<{ visual: ChatVisual }>();
const { t } = useI18n();

const title = computed(() => {
  if (visual.type === "container_ranking") return t(`cloud-chat.visual.top-${visual.metric ?? "cpu"}`);
  if (visual.type === "pattern_count") return `${visual.container} · "${visual.query}"`;
  return visual.container ?? "";
});

const facts = computed(() => (visual.facts ? factsLine(visual.facts) : ""));
// "Nothing recorded" is only true where cloud stores events for the container
// at all; without an event alert there is nothing to count, which is a
// different sentence.
const eventsLine = computed(() => {
  if (facts.value) return facts.value;
  if (visual.facts && !visual.facts.eventsRecorded) return t("cloud-chat.visual.no-events-stored");
  return t("cloud-chat.visual.no-events");
});
const limit = computed(() => visual.memoryLimit ?? 0);
const bars = computed(() => memoryBars(visual.points ?? [], limit.value));

// The one number the picture is about, at the end of its title line.
const aside = computed(() => {
  if (visual.type === "pattern_count" && visual.total != null)
    return t("cloud-chat.visual.matches", { n: visual.total.toLocaleString() });
  if (visual.type === "memory_vs_limit" && visual.facts?.memoryPeak) {
    const peak = formatBytes(visual.facts.memoryPeak, { decimals: 1 });
    return limit.value
      ? t("cloud-chat.visual.peak-of-limit", { peak, limit: formatBytes(limit.value, { decimals: 1 }) })
      : t("cloud-chat.visual.peak", { peak });
  }
  return "";
});

const rankMax = computed(() => Math.max(1e-9, ...(visual.rows ?? []).map((r) => r.value)));
function rankValue(value: number) {
  if (visual.metric === "memory") return formatBytes(value, { decimals: 1 });
  if (visual.metric === "cpu") return `${value.toFixed(1)}%`;
  return value.toLocaleString();
}
</script>
