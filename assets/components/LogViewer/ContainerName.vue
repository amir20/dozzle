<template>
  <Tag size="small" class="random-color !block w-40 truncate">
    {{ containerNames[id] }}
  </Tag>
</template>
<script lang="ts" setup>
const containerStore = useContainerStore();
const { containerNames } = storeToRefs(containerStore);

const { id } = defineProps<{
  id: string;
}>();

const colors = ["#FF0000", "#FF7F00", "#FFFF00", "#00FF00", "#00FFFF", "#0000FF", "#FF00FF", "#FF007F"];

const color = computed(() => {
  return colors[id.charCodeAt(0) % colors.length];
});
</script>

<style lang="postcss" scoped>
.random-color {
  background-color: v-bind(color);
}
</style>
