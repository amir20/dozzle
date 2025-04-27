<template>
  <LogItem :logEntry>
    <div class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></div>
  </LogItem>
</template>
<script lang="ts" setup>
import { ContainerEventLogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: ContainerEventLogEntry;
  showContainerName?: boolean;
}>();
</script>

<style scoped>
@reference "@/main.css";
[data-event="container-stopped"] {
  @apply text-red;
}
[data-event="container-started"] {
  @apply text-green;
}
</style>
