<template>
  <div>
    <span class="font-light capitalize"> RUNNING </span>
    <span class="font-semibold">
      <DistanceTime :date="container.created" strict :suffix="false" />
    </span>
  </div>
  <div>
    <span class="font-light capitalize"> Load </span>
    <span class="font-semibold"> {{ container.stat.cpu.toFixed(2) }}% </span>
  </div>
  <div>
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
