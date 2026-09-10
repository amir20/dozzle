<template>
  <LogItem :logEntry>
    <div class="cloud-event flex w-full min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 py-0.5 font-sans">
      <span class="chip">
        <component :is="icon" class="size-3" />
        {{ $t(label) }}
      </span>
      <span class="opacity-70">{{ event.detail }}</span>
      <!-- Held back, like a suppressed log event. Non-log notifications only
           reach the stream when they were held back, so this is always true
           today — kept explicit so the row does not quietly start lying if
           delivered ones are shown later. -->
      <mdi:bell-off v-if="event.suppressed" class="size-3 shrink-0 opacity-40" />
    </div>
  </LogItem>
</template>

<script lang="ts" setup>
import { CloudEventLogEntry } from "@/models/LogEntry";
import MdiSpeedometer from "~icons/mdi/speedometer";
import MdiCubeOutline from "~icons/mdi/cube-outline";

const { logEntry } = defineProps<{
  logEntry: CloudEventLogEntry;
  showContainerName?: boolean;
}>();

const event = computed(() => logEntry.event);

// Distinct icons per type, because "a threshold was crossed" and "the
// container restarted" are different kinds of news and a single generic marker
// would flatten them.
const icon = computed(() => (event.value.type === "metric" ? MdiSpeedometer : MdiCubeOutline));
const label = computed(() => (event.value.type === "metric" ? "label.event-metric" : "label.event-container"));
</script>

<style scoped>
@reference "@/main.css";
.chip {
  background-color: transparent;
  color: var(--color-info);
  border: 1px solid color-mix(in oklab, var(--color-info) 45%, transparent);
  @apply inline-flex shrink-0 items-center gap-1 rounded-xs px-1.5 py-px text-[0.62rem] font-bold tracking-wider uppercase;
}
</style>
