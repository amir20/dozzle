<template>
  <div :data-level="level" :data-position="position" class="mt-1.5 size-2.5 flex-none rounded-lg"></div>
</template>
<script lang="ts" setup>
import { Position } from "@/models/LogEntry";

defineProps<{
  level?: string;
  position?: Position;
}>();
</script>

<style scoped>
@reference "@/main.css";
[data-position="start"] {
  border-radius: 0.5em 0.5em 0 0;
  height: 70%;
  margin-bottom: -0.4em;
  margin-top: auto;
  align-self: flex-end;
}

[data-position="middle"] {
  border-radius: 0;
  height: auto;
  margin: -0.4em 0;
  align-self: stretch;
}

[data-position="end"] {
  border-radius: 0 0 0.5em 0.5em;
  height: 70%;
  margin-top: -0.4em;
  align-self: flex-start;
}
</style>
<style>
@reference "@/main.css";
[data-level="debug"],
[data-level="trace"] {
  @apply !bg-purple;
}

[data-level="info"] {
  @apply !bg-green;
}

[data-level="error"],
[data-level="fatal"] {
  @apply !bg-red;
}

[data-level="warn"] {
  @apply !bg-orange;
}
</style>
