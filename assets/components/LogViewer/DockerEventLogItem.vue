<template>
  <span class="text" :data-event="logEntry.event" v-html="logEntry.message"></span>
</template>
<script lang="ts" setup>
import { DockerEventLogEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: DockerEventLogEntry;
}>();
</script>

<style lang="scss" scoped>
span {
  &[data-event="container-stopped"] {
    color: #f14668;
  }
  &[data-event="container-started"] {
    color: hsl(141, 53%, 53%);
  }
  &.text {
    white-space: pre-wrap;
  }
}
</style>
