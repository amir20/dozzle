<template>
  <div class="is-flex-grow-1 has-text-centered my-4">
    <div class="is-relative">
      <zig-zag class="is-overlay mt-2"></zig-zag>
      <span class="text is-relative py-2 px-4">{{ $t("error.logs-skipped", { total: logEntry.totalSkipped }) }}</span>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { SkippedLogsEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: SkippedLogsEntry;
}>();
</script>

<style lang="scss" scoped>
.text {
  font-weight: bold;
  white-space: pre-wrap;
  background-color: var(--body-background-color);
}
</style>
