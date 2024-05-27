<template>
  <div class="grid w-40 overflow-hidden rounded text-center text-sm text-white">
    <div class="random-color col-start-1 row-start-1 brightness-75"></div>
    <div class="col-start-1 row-start-1 truncate px-2 brightness-100 [direction:rtl]">{{ containerNames[id] }}</div>
  </div>
</template>
<script lang="ts">
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
</script>
<script lang="ts" setup>
const containerStore = useContainerStore();
const { containerNames } = storeToRefs(containerStore);

const { id } = defineProps<{
  id: string;
}>();

const color = computed(() => colors[Math.abs(hashCode(id)) % colors.length]);
</script>

<style lang="postcss" scoped>
.random-color {
  background-color: v-bind(color);
}
</style>
