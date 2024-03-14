<template>
  <div class="relative flex w-full items-start gap-x-2">
    <log-std :std="logEntry.std" v-if="showStd" />
    <log-date :date="logEntry.date" v-if="showTimestamp" />
    <log-level class="flex" :level="logEntry.level" :position="logEntry.position" />
    <div
      class="whitespace-pre-wrap [word-break:break-word] group-[.disable-wrap]:whitespace-nowrap"
      v-html="colorize(logEntry.message)"
    ></div>
    <log-message-actions
      class="duration-250 absolute -right-1 opacity-0 transition-opacity delay-150 group-hover/entry:opacity-100"
      :message="() => decodeXML(stripAnsi(logEntry.message))"
      :log-entry="logEntry"
    />
  </div>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";
import { decodeXML } from "entities";
import AnsiConvertor from "ansi-to-html";
import stripAnsi from "strip-ansi";

const ansiConvertor = new AnsiConvertor({ escapeXML: false, fg: "var(--base-content-color)" });

defineProps<{
  logEntry: SimpleLogEntry;
}>();

const { markSearch } = useSearchFilter();
const colorize = (value: string) => markSearch(ansiConvertor.toHtml(value));
</script>
