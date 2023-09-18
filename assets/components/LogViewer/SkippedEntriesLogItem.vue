<template>
  <div class="my-4 flex-1 text-center">
    <div class="relative">
      <zig-zag class="absolute inset-0 mt-2"></zig-zag>
      <span class="bg-scheme-main-bis relative whitespace-pre-wrap px-4 py-2 font-bold">{{
        $t("error.logs-skipped", { total: logEntry.totalSkipped })
      }}</span>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { SkippedLogsEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: SkippedLogsEntry;
}>();
</script>
