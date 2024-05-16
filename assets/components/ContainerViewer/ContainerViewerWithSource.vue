<template>
  <ContainerEventSource ref="source" #default="{ messages }" @loading-more="loadingMore($event)">
    <LogViewer :messages="messages" :visible-keys="visibleKeys" :show-container-name="false" />
  </ContainerEventSource>
</template>

<script lang="ts" setup>
import LogEventSource from "@/components/ContainerViewer/LogEventSource.vue";

const { container } = useContainerContext();

const visibleKeys = persistentVisibleKeysForContainer(container);

const loadingMore = defineEmit<[value: boolean]>();

const source = $ref<InstanceType<typeof LogEventSource>>();

defineExpose({
  clear: () => source?.clear(),
});
</script>
