<template>
  <tag size="small" :std="std">
    {{ std }}
  </tag>
</template>

<script lang="ts" setup>
import { Std } from "@/models/LogEntry";

defineProps<{
  std: Std;
}>();
</script>

<style lang="scss" scoped>
.tag {
  &[std="stdout"] {
    color: var(--blue-color);
  }

  &[std="stderr"] {
    color: var(--red-color);
  }
}
</style>
