<template>
  <span class="text" v-html="colorize(logEntry.message)"></span>
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
  &::before {
    content: " ";
  }
}
</style>
