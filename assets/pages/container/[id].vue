<template>
  <search></search>
  <log-container :id="id" show-title :scrollable="activeContainers.length > 0"> </log-container>
</template>

<script lang="ts" setup>
const store = useContainerStore();
const { id } = defineProps<{ id: string }>();

const currentContainer = store.currentContainer($$(id));
const { activeContainers } = storeToRefs(store);

setTitle("loading");

onMounted(() => {
  setTitle(currentContainer.value?.name);
});

watchEffect(() => setTitle(currentContainer.value?.name));
</script>
