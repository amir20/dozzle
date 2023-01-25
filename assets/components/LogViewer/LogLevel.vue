<template>
  <div :class="level" :data-position="position"></div>
</template>
<script lang="ts" setup>
import { Position } from "@/models/LogEntry";

defineProps<{
  level?: string;
  position?: Position;
}>();
</script>

<style lang="scss" scoped>
div {
  display: inline-block;
  width: 0.7em;
  height: 0.7em;
  border-radius: 0.5em;

  &[data-position="middle"] {
    border-radius: 0;
    height: 2em;
    margin: -0.5em 0;
  }

  &[data-position="start"] {
    border-radius: 0.5em 0.5em 0 0;
    height: 1.2em;
    margin-bottom: -0.4em;
  }

  &[data-position="end"] {
    border-radius: 0 0 0.5em 0.5em;
    height: 1.4em;
    margin-top: -0.4em;
  }

  &.debug,
  &.trace {
    background-color: #9c27b0;
  }

  &.info {
    background-color: #00b5ad;
  }

  &.error,
  &.fatal {
    background-color: #f44336;
  }

  &.warn {
    background-color: #ff9800;
  }
}
</style>
