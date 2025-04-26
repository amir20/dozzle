<template>
  <div class="inline-flex size-4" :health="health" v-if="health" :title="health">
    <cil:check-circle v-if="health == 'healthy'" />
    <cil:x-circle v-else-if="health == 'unhealthy'" />
    <cil:circle v-else />
  </div>
</template>

<script lang="ts" setup>
import { ContainerHealth } from "@/types/Container";

defineProps<{
  health: ContainerHealth | undefined;
}>();
</script>

<style scoped>
@reference "@/main.css";
[health="unhealthy"] {
  @apply text-red;
}

[health="healthy"] {
  @apply text-green;
}
</style>
