<template>
  <div :std="std" class="pt-px text-[0.85em] tracking-wide uppercase opacity-70">
    {{ std }}
  </div>
</template>

<script lang="ts" setup>
import { Std } from "@/models/LogEntry";

defineProps<{
  std: Std;
}>();
</script>

<style scoped>
@reference "@/main.css";
[std="stdout"] {
  @apply text-blue;
}

[std="stderr"] {
  @apply text-red;
}
</style>
