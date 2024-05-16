<template>
  <StackEventSource ref="source" #default="{ messages }" @loading-more="loadingMore($event)">
    <LogViewer :messages="messages" :visible-keys="visibleKeys" :show-container-name="true" />
  </StackEventSource>
</template>

<script lang="ts" setup>
import StackEventSource from "./StackEventSource.vue";

const visibleKeys: Ref<string[][]> = ref([]);

const loadingMore = defineEmit<[value: boolean]>();
const source = $ref<InstanceType<typeof StackEventSource>>();
defineExpose({
  clear: () => source?.clear(),
});
</script>
