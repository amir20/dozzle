<template>
  <div class="host">
    <div class="parent">
      <zig-zag class="zigzag"></zig-zag>
      <span class="text">{{ $t("error.logs-skipped", { total: logEntry.totalSkipped }) }}</span>
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
.parent {
  position: relative;
}
.host {
  flex-grow: 1;
  text-align: center;
  margin: 1em 0;
}
.text {
  font-weight: bold;
  white-space: pre-wrap;
  position: relative;
  background-color: var(--body-background-color);
  padding: 0.5em 1em;
}

.zigzag {
  position: absolute;
  left: 0;
  top: 50%;
}
</style>
