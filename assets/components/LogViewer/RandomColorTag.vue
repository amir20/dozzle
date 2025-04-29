<template>
  <div class="tag grid overflow-hidden rounded-sm text-center text-sm text-white">
    <div class="random-color col-start-1 row-start-1 brightness-75"></div>
    <div class="col-start-1 row-start-1 truncate px-2 brightness-100" :class="truncateRight ? '[direction:rtl]' : ''">
      {{ value }}
    </div>
  </div>
</template>
<script lang="ts">
const colors = [
  "hsl(200, 85%, 65%)", // Vibrant Sky Blue
  "hsl(150, 85%, 65%)", // Vibrant Mint
  "hsl(300, 85%, 65%)", // Vibrant Purple
  "hsl(25, 85%, 65%)", // Vibrant Peach
  "hsl(270, 85%, 65%)", // Vibrant Lavender
  "hsl(340, 85%, 65%)", // Vibrant Rose
  "hsl(170, 85%, 65%)", // Vibrant Aqua
  "hsl(50, 85%, 65%)", // Vibrant Yellow
  "hsl(235, 85%, 65%)", // Vibrant Periwinkle
  "hsl(10, 85%, 65%)", // Vibrant Coral
  "hsl(180, 85%, 65%)", // Vibrant Turquoise
  "hsl(320, 85%, 65%)", // Vibrant Orchid
  "hsl(90, 85%, 65%)", // Vibrant Lime
  "hsl(260, 85%, 65%)", // Vibrant Amethyst
  "hsl(30, 85%, 65%)", // Vibrant Orange
  "hsl(210, 85%, 65%)", // Vibrant Ocean
  "hsl(290, 85%, 65%)", // Vibrant Mauve
  "hsl(120, 85%, 65%)", // Vibrant Spring
  "hsl(350, 85%, 65%)", // Vibrant Strawberry
  "hsl(190, 85%, 65%)", // Vibrant Azure
] as const;
</script>
<script lang="ts" setup>
const { value, truncateRight = false } = defineProps<{
  value: string;
  truncateRight?: boolean;
}>();

const color = computed(() => colors[Math.abs(hashCode(value)) % colors.length]);
</script>

<style scoped>
.random-color {
  background-color: v-bind(color);
}
</style>
