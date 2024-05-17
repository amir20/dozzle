<template>
  <InfiniteLoader :onLoadMore="fetchMore" :enabled="messages.length > 100"></InfiniteLoader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup>
import { LogStreamSource } from "@/composable/eventStreams";

const loadingMore = defineEmit<[value: boolean]>();

const { streamSource } = defineProps<{
  streamSource: () => LogStreamSource;
}>();

const { messages, loadOlderLogs } = streamSource();

const beforeLoading = () => loadingMore(true);
const afterLoading = () => loadingMore(false);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = () => loadOlderLogs({ beforeLoading, afterLoading });
</script>
