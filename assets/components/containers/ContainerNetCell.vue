<template>
  <!-- Fixed width with the figures right-aligned: the value changes every second, and
       a cell that sized itself to it would shove every column to its left around. -->
  <div v-if="rate === undefined" class="text-base-content/40 w-28">&ndash;</div>
  <div v-else class="flex w-28 flex-col font-mono text-xs leading-tight whitespace-nowrap tabular-nums">
    <span class="flex justify-between gap-2" :class="{ 'text-base-content/40': !rate.rx }">
      <span>↓</span>{{ format(rate.rx) }}
    </span>
    <span class="flex justify-between gap-2" :class="{ 'text-base-content/40': !rate.tx }">
      <span>↑</span>{{ format(rate.tx) }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { networkRate, type Container } from "@/models/Container";

const { container } = defineProps<{
  container: Container;
}>();

const rate = computed(() => networkRate(container));

// "B" rather than formatBytes' "Bytes", which alone would need a wider column
const format = (bytes: number) => (bytes ? formatBytes(bytes, { decimals: 1 }).replace(" Bytes", " B") : "0 B") + "/s";
</script>
