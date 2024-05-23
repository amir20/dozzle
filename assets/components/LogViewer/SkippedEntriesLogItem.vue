<template>
  <div class="my-4 flex-1 text-center">
    <div class="relative">
      <ZigZag class="absolute inset-0 mt-2" />
      <span class="relative whitespace-pre-wrap bg-base px-4 py-2 font-bold">
        {{ $t("error.logs-skipped", { total: logEntry.totalSkipped }) }}
      </span>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { SkippedLogsEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: SkippedLogsEntry;
}>();
</script>
