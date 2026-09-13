<template>
  <LogItem :logEntry>
    <LogLevel class="flex select-none" :level="logEntry.level" :event="logEntry.matchedEvent" />
    <div
      class="log-message [word-break:break-word] whitespace-pre-wrap group-[.disable-wrap]:whitespace-pre"
      v-html="colorize(displayed)"
    ></div>
  </LogItem>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";
import { showTimestamp } from "@/stores/settings";

const { logEntry } = defineProps<{
  logEntry: SimpleLogEntry;
}>();

// The app's own timestamp only repeats the date column, so it goes when that
// column is on. With the column off it is the only time left on the line.
const displayed = computed(() =>
  showTimestamp.value ? logEntry.message.slice(logEntry.timestampPrefix) : logEntry.message,
);
</script>
