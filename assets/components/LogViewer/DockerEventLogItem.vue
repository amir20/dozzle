<template>
  <span class="whitespace-pre-wrap" :data-event="logEntry.event" v-html="logEntry.message"></span>
</template>
<script lang="ts" setup>
import { DockerEventLogEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: DockerEventLogEntry;
}>();
</script>

<style lang="postcss" scoped>
[data-event="container-stopped"] {
  @apply text-red;
}
[data-event="container-started"] {
  @apply text-green;
}
</style>
