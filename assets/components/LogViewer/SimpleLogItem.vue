<template>
  <LogItem :logEntry>
    <LogLevel class="flex select-none" :level="logEntry.level" />
    <div
      class="[word-break:break-word] whitespace-pre-wrap group-[.disable-wrap]:whitespace-pre"
      v-html="colorize(logEntry.message)"
    ></div>
  </LogItem>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: SimpleLogEntry;
}>();
</script>
