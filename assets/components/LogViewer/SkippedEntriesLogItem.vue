<template>
  <span class="text">{{ $t("error.logs-skipped", { total: logEntry.totalSkipped }) }}</span>
</template>
<script lang="ts" setup>
import { SkippedLogsEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: SkippedLogsEntry;
}>();
</script>

<style lang="scss" scoped>
span {
  &.text {
    flex-grow: 1;
    text-align: center;
    font-weight: bold;
    white-space: pre-wrap;
    &::before {
      content: " ";
    }
  }
}
</style>
