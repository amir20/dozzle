<template>
  <LogItem :logEntry>
    <div
      class="log-wrapper [word-break:break-word] whitespace-pre-wrap group-[.disable-wrap]:whitespace-nowrap"
      v-html="linkify(colorize(logEntry.message))"
    ></div>
    <LogMessageActions
      class="absolute -right-1 opacity-0 transition-opacity delay-150 duration-250 group-hover/entry:opacity-100"
      :message="() => decodeXML(stripAnsi(logEntry.message))"
      :log-entry="logEntry"
    />
  </LogItem>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";
import { decodeXML } from "entities";
import AnsiConvertor from "ansi-to-html";
import stripAnsi from "strip-ansi";

const ansiConvertor = new AnsiConvertor({
  escapeXML: false,
  fg: "var(--color-base-content)",
  bg: "var(--color-base-100)",
});

defineProps<{
  logEntry: SimpleLogEntry;
}>();

const colorize = (value: string) => ansiConvertor.toHtml(value);
const urlPattern =
  /https?:\/\/(?:www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b[-a-zA-Z0-9()@:%_+.~#?&\/=]*/g;
const linkify = (text: string) =>
  text.replace(urlPattern, (url) => `<a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a>`);
</script>

<style scoped>
@import "@/main.css" reference;
.log-wrapper :deep(a) {
  @apply text-primary underline-offset-4 hover:underline;
}
</style>
