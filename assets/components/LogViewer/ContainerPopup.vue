<template>
  <div>
    <span class="has-text-weight-light"> RUNNING </span>
    <span class="has-text-weight-semibold">
      <distance-time :date="container.created" strict :suffix="false"></distance-time>
    </span>
  </div>
  <div>
    <span class="has-text-weight-light"> LOAD </span>
    <span class="has-text-weight-semibold"> {{ container.getLastStat().snapshot.cpu }}% </span>
  </div>
  <div>
    <span class="has-text-weight-light"> MEM </span>
    <span class="has-text-weight-semibold"> {{ formatBytes(container.getLastStat().snapshot.memoryUsage) }} </span>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{
  container: Container;
}>();
</script>

<style lang="scss" scoped></style>
