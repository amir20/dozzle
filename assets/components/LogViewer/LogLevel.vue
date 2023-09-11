<template>
  <div :data-level="level" :data-position="position" class="inline-block"></div>
</template>
<script lang="ts" setup>
import { Position } from "@/models/LogEntry";

defineProps<{
  level?: string;
  position?: Position;
}>();
</script>

<style lang="postcss" scoped>
div {
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
    margin: -0.4em 0;
    align-self: auto;
  }

  &[data-position="end"] {
    border-radius: 0 0 0.5em 0.5em;
    height: 70%;
    margin-top: -0.3em;
    align-self: flex-start;
  }

  &[data-level="debug"],
  &[data-level="trace"] {
    @apply bg-purple;
  }

  &[data-level="info"] {
    @apply bg-green;
  }

  &[data-level="error"],
  &[data-level="fatal"] {
    @apply bg-red;
  }

  &[data-level="warn"],
  &[data-level="warning"] {
    @apply bg-orange;
  }
}
</style>
