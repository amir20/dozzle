<template>
  <EventSource ref="source" #default="{ messages }" @loading-more="loadingMore($event)" :stream-source="streamSource">
    <LogViewer :messages="messages" :visible-keys="visibleKeys" :show-container-name="false" />
  </EventSource>
</template>

<script lang="ts" setup>
import LogEventSource from "@/components/ContainerViewer/LogEventSource.vue";
import { LogStreamSource } from "@/composable/eventStreams";

const { streamSource, visibleKeys } = defineProps<{
  streamSource: () => LogStreamSource;
  visibleKeys: string[][];
}>();

const loadingMore = defineEmit<[value: boolean]>();

const source = $ref<InstanceType<typeof LogEventSource>>();

defineExpose({
  clear: () => source?.clear(),
});
</script>
