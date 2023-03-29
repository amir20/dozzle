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
  align-self: auto;
  margin-top: 0.4em;

  &[data-position="start"] {
    border-radius: 0.5em 0.5em 0 0;
    height: 70%;
    margin-bottom: -0.2em;
    margin-top: auto;
    align-self: flex-end;
  }

  &[data-position="middle"] {
    border-radius: 0;
    height: auto;
    margin: -0.2em 0;
    align-self: auto;
  }

  &[data-position="end"] {
    border-radius: 0 0 0.5em 0.5em;
    height: 70%;
    margin-top: -0.2em;
    align-self: flex-start;
  }

  &.debug,
  &.trace {
    background-color: var(--purple-color);
  }

  &.info {
    background-color: var(--green-color);
  }

  &.error,
  &.fatal {
    background-color: var(--red-color);
  }

  &.warn {
    background-color: var(--orange-color);
  }
}
</style>
