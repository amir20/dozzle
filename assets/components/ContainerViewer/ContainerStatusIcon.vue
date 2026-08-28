<template>
  <div class="inline-flex size-4" :data-status="status" :title="status">
    <cil:check-circle v-if="health === 'healthy'" />
    <cil:x-circle v-else-if="health === 'unhealthy'" />
    <cil:media-pause v-else-if="state === 'paused'" />
    <cil:circle v-else />
  </div>
</template>

<script lang="ts" setup>
import { ContainerHealth, ContainerState } from "@/types/Container";

const { state, health } = defineProps<{
  state: ContainerState;
  health?: ContainerHealth;
}>();

const status = computed(() => health ?? state);
</script>

<style scoped>
@reference "@/main.css";

[data-status="unhealthy"],
[data-status="exited"],
[data-status="dead"] {
  @apply text-red;
}

[data-status="healthy"],
[data-status="running"] {
  @apply text-green;
}

[data-status="starting"],
[data-status="restarting"],
[data-status="paused"] {
  @apply text-orange;
}

[data-status="created"],
[data-status="deleted"] {
  @apply text-base-content/40;
}
</style>
