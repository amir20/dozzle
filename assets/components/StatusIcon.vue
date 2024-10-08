<template>
  <span :title="state" :class="state" class="mr-2">
    <Mdi:chevron-down-circle-outline v-if="state == 'created'" class="text-base-contenty inline" />
    <Mdi:circle-slice-8 v-if="state == 'running'" class="inline text-green" />
    <Mdi:stop-circle-outline v-if="state == 'exited'" class="text-base-contenty inline" />
    <Mdi:alert-circle v-if="state == 'dead'" class="inline text-red" />
    <Mdi:pause-circle-outline v-if="state == 'paused'" class="inline text-secondary" />
    <Mdi:sync v-if="state == 'restarting'" class="inline text-orange" />
    <Mdi:delete-circle-outline v-if="state == 'deleted'" class="inline text-base-content" />
  </span>
</template>

<script lang="ts" setup>
import { ContainerState } from "@/types/Container";

const { state } = defineProps<{
  state: ContainerState;
}>();
</script>

<style lang="postcss" scoped>
.break-world {
  overflow-wrap: break-word;
}
</style>
