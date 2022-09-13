<template>
  <span class="text">{{ logEntry.message }}</span>
</template>
<script lang="ts" setup>
import { DockerEventLogEntry } from "@/models/LogEntry";

defineProps<{
  logEntry: DockerEventLogEntry;
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
