<template>
  <div class="flex items-start gap-x-2">
    <log-std :std="logEntry.std" v-if="showStd" />
    <log-date :date="logEntry.date" v-if="showTimestamp" />
    <log-level class="flex" :level="logEntry.level" :position="logEntry.position" />
    <div class="whitespace-pre-wrap group-[.disable-wrap]:whitespace-nowrap" v-html="colorize(logEntry.message)"></div>
  </div>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";
import AnsiConvertor from "ansi-to-html";

const ansiConvertor = new AnsiConvertor({ escapeXML: false, fg: "var(--text-color)" });
defineProps<{
  logEntry: SimpleLogEntry;
}>();

const { markSearch } = useSearchFilter();
const colorize = (value: string) => markSearch(ansiConvertor.toHtml(value));
</script>
