<template>
  <mdi:satellite-variant v-if="type == 'agent'" />
  <ph:globe-simple v-else-if="type == 'remote'" />
  <mdi:hexagon-multiple v-else-if="type == 'swarm'" />
  <ph:computer-tower v-else />
</template>
<script setup lang="ts">
import { Host } from "@/stores/hosts";
const { type } = defineProps<{ type: Host["type"] }>();
</script>
