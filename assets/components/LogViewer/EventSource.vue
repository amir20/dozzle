<template>
  <InfiniteLoader :onLoadMore="fetchMore" :enabled="messages.length > 50"></InfiniteLoader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";

const loadingMore = defineEmit<[value: boolean]>();

const { entity, streamSource } = $defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { messages, loadOlderLogs } = streamSource($$(entity));

const beforeLoading = () => loadingMore(true);
const afterLoading = () => loadingMore(false);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = async () => {
  beforeLoading();
  await loadOlderLogs();
  afterLoading();
};
</script>
