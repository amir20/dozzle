<template>
  <EventSource ref="source" #default="{ messages }" @loading-more="loadingMore($event)" :stream-source="streamSource">
    <LogViewer :messages="messages" :visible-keys="visibleKeys" :show-container-name="showContainerName" />
  </EventSource>
</template>

<script lang="ts" setup>
import LogEventSource from "@/components/ContainerViewer/LogEventSource.vue";
import { LogStreamSource } from "@/composable/eventStreams";

const { streamSource, visibleKeys, showContainerName } = defineProps<{
  streamSource: () => LogStreamSource;
  visibleKeys: string[][];
  showContainerName: boolean;
}>();

const loadingMore = defineEmit<[value: boolean]>();

const source = $ref<InstanceType<typeof LogEventSource>>();

defineExpose({
  clear: () => source?.clear(),
});
</script>
