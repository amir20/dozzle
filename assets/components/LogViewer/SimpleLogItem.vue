<template>
  <div
    class="flex items-start gap-x-2 hover:cursor-pointer"
    @mouseover="isHovering = true"
    @mouseleave="isHovering = false"
  >
    <log-std :std="logEntry.std" v-if="showStd" />
    <log-date :date="logEntry.date" v-if="showTimestamp" />
    <log-level class="flex" :level="logEntry.level" :position="logEntry.position" />
    <div class="whitespace-pre-wrap group-[.disable-wrap]:whitespace-nowrap" v-html="colorize(logEntry.message)"></div>
    <div
      v-show="isHovering"
      class="rounded px-1 py-1 hover:bg-slate-700 hover:text-secondary"
      @click="logEntry.copyLogMessageToClipBoard()"
    >
      <carbon:copy-file />
    </div>
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
const isHovering = ref(false);
</script>
