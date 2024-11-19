<template>
  <LogItem :logEntry>
    <div
      class="log-wrapper whitespace-pre-wrap [word-break:break-word] group-[.disable-wrap]:whitespace-nowrap"
      v-html="linkify(colorize(logEntry.message))"
    ></div>
    <LogMessageActions
      class="duration-250 absolute -right-1 opacity-0 transition-opacity delay-150 group-hover/entry:opacity-100"
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
  fg: "oklch(var(--base-content-color))",
  bg: "oklch(var(--base-color))",
});

defineProps<{
  logEntry: SimpleLogEntry;
}>();

const colorize = (value: string) => ansiConvertor.toHtml(value);
const urlPattern = /(https?:\/\/[^\s]+)/g;
const linkify = (text: string) =>
  text.replace(urlPattern, (url) => `<a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a>`);
</script>
