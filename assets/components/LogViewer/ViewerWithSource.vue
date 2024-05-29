<template>
  <EventSource
    ref="source"
    #default="{ messages }"
    @loading-more="loadingMore($event)"
    :stream-source="streamSource"
    :entity="entity"
  >
    <LogViewer :messages="messages" :visible-keys="visibleKeys" :show-container-name="showContainerName" />
  </EventSource>
</template>

<script lang="ts" setup generic="T">
import LogEventSource from "@/components/ContainerViewer/LogEventSource.vue";
import { LogStreamSource } from "@/composable/eventStreams";

const { streamSource, visibleKeys, showContainerName, entity } = defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  visibleKeys: string[][];
  showContainerName: boolean;
  entity: T;
}>();

const loadingMore = defineEmit<[value: boolean]>();

const source = $ref<InstanceType<typeof LogEventSource>>();

defineExpose({
  clear: () => source?.clear(),
});

onKeyStroke("k", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
    source?.clear();
    e.preventDefault();
  }
});
</script>
