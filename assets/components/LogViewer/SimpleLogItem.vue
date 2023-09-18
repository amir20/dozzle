<template>
  <div class="flex gap-x-2">
    <div v-if="showStd">
      <log-std :std="logEntry.std" />
    </div>
    <div v-if="showTimestamp">
      <log-date :date="logEntry.date" />
    </div>
    <div class="flex">
      <log-level :level="logEntry.level" :position="logEntry.position" />
    </div>
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
