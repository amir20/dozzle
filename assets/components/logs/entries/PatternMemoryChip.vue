<template>
  <!-- Error memory from Dozzle Cloud: says something only when there is
       something to say. Known patterns, and memory still learning, render
       nothing at all. -->
  <button
    v-if="memory && SHOWN_STATUSES.has(memory.status)"
    type="button"
    class="mt-0.5 inline-flex shrink-0 select-none"
    :title="tooltip"
    @mouseenter="hovered"
    @click.stop="openWhereItStarted"
  >
    <MemoryNewChip />
  </button>
</template>

<script lang="ts" setup>
import type { LogEntry, LogMessage, PatternMemory } from "@/models/LogEntry";
import { SHOWN_STATUSES, chipMoment } from "@/composable/cloud/patternMemory";

const { memory, logEntry } = defineProps<{ memory?: PatternMemory; logEntry: LogEntry<LogMessage> }>();

const { t, locale } = useI18n();
const { resetSearch } = useSearchFilter();
const { jumpTo } = useLogJump();

const tooltip = computed(() => {
  if (!memory?.firstSeen) return t("tooltip.memory-new");
  const when = new Date(memory.firstSeen / 1_000_000).toLocaleString(locale.value, {
    dateStyle: "medium",
    timeStyle: "short",
  });
  return t("tooltip.memory-new-since", { time: when });
});

// Counted once per chip, so the number is "chips someone looked at", not
// "times a pointer crossed one".
let counted = false;
function hovered() {
  if (counted) return;
  counted = true;
  trackUsage("memory.chip.hover");
}

function openWhereItStarted() {
  if (!memory) return;
  trackUsage("memory.chip.open");
  resetSearch();
  jumpTo(chipMoment(memory, logEntry));
}
</script>
