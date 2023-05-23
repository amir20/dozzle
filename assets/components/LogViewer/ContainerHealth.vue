<template>
  <div class="icon is-small health" :health="health" v-if="health" :title="health">
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

<style lang="scss" scoped>
.health {
  &[health="unhealthy"] {
    color: var(--red-color);
  }

  &[health="healthy"] {
    color: var(--green-color);
  }
}
</style>
