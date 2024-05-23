<template>
  <InfiniteLoader :onLoadMore="fetchMore" :enabled="messages.length > 50"></InfiniteLoader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";

const loadingMore = defineEmit<[value: boolean]>();

const props = defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { entity, streamSource } = toRefs(props);

const { messages, loadOlderLogs } = streamSource.value(entity);

const beforeLoading = () => loadingMore(true);
const afterLoading = () => loadingMore(false);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = () => loadOlderLogs({ beforeLoading, afterLoading });
</script>
