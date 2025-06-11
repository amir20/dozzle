<template>
  <LogItem :logEntry>
    <div
      class="[word-break:break-word] whitespace-pre-wrap group-[.disable-wrap]:whitespace-nowrap"
      v-html="colorize(logEntry.message)"
    ></div>
  </LogItem>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";
import AnsiConvertor from "ansi-to-html";

const ansiConvertor = new AnsiConvertor({
  escapeXML: false,
  fg: "var(--color-base-content)",
  bg: "var(--color-base-100)",
});

defineProps<{
  logEntry: SimpleLogEntry;
}>();

const colorize = (value: string) => ansiConvertor.toHtml(value);
</script>
