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

const { containers } = useLoggingContext();

const colors = [
  "#4B0082",
  "#FF00FF",
  "#FF7F00",
  "#FFFF00",
  "#00FF00",
  "#00FFFF",
  "#FF0000",
  "#0000FF",
  "#FF007F",
  "#32CD32",
  "#40E0D0",
  "#E6E6FA",
  "#800080",
  "#FFD700",
  "#FF4040",
] as const;

const color = computed(() => {
  const index = containers.value.findIndex((container) => container.id === id);
  return colors[index % colors.length];
});
</script>

<style lang="postcss" scoped>
.random-color {
  background-color: v-bind(color);
}

.direction-rtl {
  direction: rtl;
}
</style>
