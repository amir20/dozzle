<template>
  <div
    :data-level="level"
    :data-position="position"
    class="mt-1.5 size-2.5 flex-none rounded-lg"
    :class="{ showUnknown }"
  ></div>
</template>
<script lang="ts" setup>
import { Position, Level } from "@/models/LogEntry";

const {
  level,
  position,
  showUnknown = false,
} = defineProps<{
  level?: Level;
  position?: Position;
  showUnknown?: boolean;
}>();
</script>

<style scoped>
@reference "@/main.css";
[data-position="start"],
[data-position="middle"],
[data-position="end"] {
  align-self: stretch;
  height: auto;
}

[data-position="start"] {
  border-radius: 0.375rem 0.375rem 0 0;
}

[data-position="middle"] {
  border-radius: 0;
  margin-top: 0;
}

[data-position="end"] {
  border-radius: 0 0 0.375rem 0.375rem;
  margin-top: 0;
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

[data-level="unknown"].show-unknown {
  @apply !bg-base-300;
}
</style>
