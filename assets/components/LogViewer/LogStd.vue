<template>
  <Tag size="small" :std="std" class="items-start!">
    {{ std }}
  </Tag>
</template>

<script lang="ts" setup>
import { Std } from "@/models/LogEntry";

defineProps<{
  std: Std;
}>();
</script>

<style scoped>
@import "@/main.css" reference;
[std="stdout"] {
  @apply text-blue;
}

[std="stderr"] {
  @apply text-red;
}
</style>
