<template>
  <div class="relative block w-40 overflow-hidden rounded px-1.5 text-center text-sm text-white">
    <div class="random-color absolute inset-0 brightness-75"></div>
    <div class="direction-rtl relative truncate">{{ containerNames[id] }}</div>
  </div>
</template>
<script lang="ts" setup>
const containerStore = useContainerStore();
const { containerNames } = storeToRefs(containerStore);

const { id } = defineProps<{
  id: string;
}>();

const colors = [
  "#FF0000",
  "#FF7F00",
  "#FFFF00",
  "#00FF00",
  "#00FFFF",
  "#0000FF",
  "#FF00FF",
  "#FF007F",
  "#32CD32",
  "#40E0D0",
  "#800080",
  "#FFD700",
  "#FF4040",
  "#4B0082",
  "#008080",
  "#E6E6FA",
];

const color = computed(() => colors[Math.abs(hashCode(id)) % colors.length]);
</script>

<style lang="postcss" scoped>
.random-color {
  background-color: v-bind(color);
}

.direction-rtl {
  direction: rtl;
}
</style>
