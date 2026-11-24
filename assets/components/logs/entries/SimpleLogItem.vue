<template>
  <LogItem :logEntry>
    <LogLevel class="flex select-none" :level="logEntry.level" :event="logEntry.matchedEvent" />
    <div
      class="log-message [word-break:break-word] whitespace-pre-wrap group-[.disable-wrap]:whitespace-pre"
      v-html="colorize(displayed)"
    ></div>
    <RepeatCount v-if="fold" :fold @toggle="$emit('toggleFold')" />
    <PatternMemoryChip v-if="logEntry.patternMemory" :memory="logEntry.patternMemory" :log-entry="logEntry" />
  </LogItem>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";
import { showTimestamp } from "@/stores/settings";
import type { FoldRun } from "@/utils/readable";

const { logEntry } = defineProps<{
  logEntry: SimpleLogEntry;
  /** Set when this row stands for a run of repeated lines. */
  fold?: FoldRun;
}>();
defineEmits<{ toggleFold: [] }>();

// The app's own timestamp only repeats the date column, so it goes when that
// column is on. With the column off it is the only time left on the line.
const displayed = computed(() =>
  showTimestamp.value ? logEntry.message.slice(logEntry.timestampPrefix) : logEntry.message,
);
</script>
