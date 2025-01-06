<template>
  <div>
    <span class="font-light capitalize"> STATE </span>
    <span class="font-semibold uppercase"> {{ container.state }} </span>
  </div>
  <div v-if="container.startedAt.getFullYear() > 0">
    <span class="font-light capitalize"> STARTED </span>
    <span class="font-semibold">
      <DistanceTime :date="container.startedAt" strict />
    </span>
  </div>
  <div v-if="container.state != 'running' && container.finishedAt.getFullYear() > 0">
    <span class="font-light capitalize"> FINISHED </span>
    <span class="font-semibold">
      <DistanceTime :date="container.finishedAt" strict />
    </span>
  </div>
  <div v-if="container.state == 'running'">
    <span class="font-light capitalize"> Load </span>
    <span class="font-semibold"> {{ container.stat.cpu.toFixed(2) }}% </span>
  </div>
  <div v-if="container.state == 'running'">
    <span class="font-light capitalize"> MEM </span>
    <span class="font-semibold"> {{ formatBytes(container.stat.memoryUsage) }} </span>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{
  container: Container;
}>();
</script>
