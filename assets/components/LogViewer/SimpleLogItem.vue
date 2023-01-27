<template>
  <div class="columns is-1 is-variable is-mobile">
    <div class="column is-narrow" v-if="showTimestamp">
      <log-date :date="logEntry.date"></log-date>
    </div>
    <div class="column is-narrow is-flex">
      <log-level :level="logEntry.level" :position="logEntry.position"></log-level>
    </div>
    <div class="text column" v-html="colorize(logEntry.message)"></div>
  </div>
</template>
<script lang="ts" setup>
import { SimpleLogEntry } from "@/models/LogEntry";
import AnsiConvertor from "ansi-to-html";

const ansiConvertor = new AnsiConvertor({ escapeXML: true });
defineProps<{
  logEntry: SimpleLogEntry;
}>();

const { markSearch } = useSearchFilter();
const colorize = (value: string) => markSearch(ansiConvertor.toHtml(value));
</script>

<style lang="scss" scoped>
.disable-wrap {
  .text {
    white-space: nowrap;
  }
}

.text {
  white-space: pre-wrap;
}
</style>
