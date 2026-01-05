<template>
  <LogItem :logEntry>
    <div class="flex flex-col">
      <div v-for="(msg, index) in logEntry.message" :key="index" class="flex items-start gap-x-2">
        <LogLevel class="flex select-none" :level="logEntry.level" :position="getPosition(index)" />
        <div
          class="[word-break:break-word] whitespace-pre-wrap group-[.disable-wrap]:whitespace-pre"
          v-html="colorize(msg)"
        ></div>
      </div>
    </div>
  </LogItem>
</template>
<script lang="ts" setup>
import { GroupedLogEntry, type Position } from "@/models/LogEntry";
import AnsiConvertor from "ansi-to-html";

const ansiConvertor = new AnsiConvertor({
  escapeXML: false,
  fg: "var(--color-base-content)",
  bg: "var(--color-base-100)",
});

const { logEntry } = defineProps<{
  logEntry: GroupedLogEntry;
}>();

const getPosition = (index: number): Position => {
  const len = logEntry.message.length;
  if (index === 0) return "start";
  if (index === len - 1) return "end";
  return "middle";
};

const colorize = (value: string) => ansiConvertor.toHtml(value);
</script>
