<template>
  <div class="grid grid-flow-row grid-cols-2 grid-rows-8 gap-y-2">
    <div class="font-light capitalize">ID</div>
    <div class="font-semibold">{{ container.id }}</div>
    <div class="font-light capitalize">{{ $t("label.status") }}</div>
    <div class="font-semibold">{{ container.state }}</div>
    <div class="font-light capitalize">{{ $t("label.host") }}</div>
    <div class="font-semibold">{{ container.hostLabel }}</div>
    <div class="font-light capitalize">{{ $t("label.image") }}</div>
    <div class="break-world font-semibold">{{ container.image }}</div>
    <div v-if="container.health" class="font-light capitalize">{{ $t("label.health") }}</div>
    <div v-if="container.health" class="font-semibold">{{ container.health }}</div>
    <div class="font-light capitalize">{{ $t("label.ports") }}</div>
    <div class="font-semibold">{{ container.ports }}</div>
    <div class="font-light capitalize">{{ $t("label.created") }}</div>
    <div class="font-semibold">
      <DistanceTime :date="container.created" strict :suffix="false" />
    </div>
    <div class="font-light capitalize">{{ $t("label.cpu-usage") }}</div>
    <div class="font-semibold">{{ container.stat.cpu.toFixed(2) }}%</div>
    <div class="font-light capitalize">{{ $t("label.mem-usage") }}</div>
    <div class="font-semibold">{{ formatBytes(container.stat.memoryUsage) }}</div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{
  container: Container;
}>();
</script>

<style scoped>
.grid {
  grid-template-rows: fit-content(100%);
}

.break-world {
  overflow-wrap: break-word;
}
</style>
