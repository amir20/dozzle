<template>
  <div class="my-4 flex-1 text-center">
    <div class="relative">
      <ZigZag class="absolute inset-0 mt-2" />
      <button
        class="btn btn-primary relative whitespace-pre-wrap"
        @click="logEntry.loadSkippedEntries()"
        @mouseover="console.log(logEntry.totalSkipped)"
      >
        {{ $t("error.logs-skipped", { total: logEntry.totalSkipped }) }}
      </button>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { SkippedLogsEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: SkippedLogsEntry;
}>();
</script>
