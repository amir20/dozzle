<template>
  <!-- Fixed width with the figures right-aligned: the value changes every second, and
       a cell that sized itself to it would shove every column to its left around. -->
  <div v-if="rate === undefined" class="text-base-content/40 w-28">&ndash;</div>
  <div v-else class="flex w-28 flex-col font-mono text-xs leading-tight whitespace-nowrap tabular-nums">
    <span class="flex justify-between gap-2" :class="{ 'text-base-content/40': rate.down < 1 }">
      <span>↓</span>{{ format(rate.down) }}
    </span>
    <span class="flex justify-between gap-2" :class="{ 'text-base-content/40': rate.up < 1 }">
      <span>↑</span>{{ format(rate.up) }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { diskRate, networkRate, type Container } from "@/models/Container";

const {
  container,
  type,
  average = false,
} = defineProps<{
  container: Container;
  // network reads ↓ received and ↑ sent, disk ↓ read and ↑ written, like IOCard
  type: "net" | "disk";
  average?: boolean;
}>();

const rate = computed(() => {
  if (type === "net") {
    const net = networkRate(container, average);
    return net && { down: net.rx, up: net.tx };
  }
  const disk = diskRate(container, average);
  return disk && { down: disk.read, up: disk.write };
});

// An average decays toward zero without reaching it, so under a byte reads as idle.
// "B" rather than formatBytes' "Bytes", which alone would need a wider column
const format = (bytes: number) =>
  (bytes >= 1 ? formatBytes(bytes, { decimals: 1 }).replace(" Bytes", " B") : "0 B") + "/s";
</script>
