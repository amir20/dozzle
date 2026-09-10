<template>
  <LogItem :logEntry>
    <div class="flex flex-col">
      <div v-for="(msg, index) in logEntry.message" :key="index" class="flex items-start gap-x-2">
        <LogLevel
          class="flex select-none"
          :level="logEntry.level"
          :position="getPosition(index)"
          :event="index === 0 ? logEntry.matchedEvent : undefined"
        />
        <div
          class="log-message [word-break:break-word] whitespace-pre-wrap group-[.disable-wrap]:whitespace-pre"
          v-html="colorize(msg)"
          :class="{ 'min-h-4': msg === '' }"
        ></div>
      </div>
    </div>
  </LogItem>
</template>
<script lang="ts" setup>
import { GroupedLogEntry, type Position } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: GroupedLogEntry;
}>();

// A one-line group is not a group as far as the marker goes: it takes the dot
// like any other single line rather than a stub of rail.
const getPosition = (index: number): Position | undefined => {
  const len = logEntry.message.length;
  if (len === 1) return undefined;
  if (index === 0) return "start";
  if (index === len - 1) return "end";
  return "middle";
};
</script>
