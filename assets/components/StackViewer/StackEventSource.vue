<template>
  <InfiniteLoader :onLoadMore="fetchMore" :enabled="messages.length > 100"></InfiniteLoader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup>
const loadingMore = defineEmit<[value: boolean]>();

const { messages, loadOlderLogs } = useStackContextLogStream();

const beforeLoading = () => loadingMore(true);
const afterLoading = () => loadingMore(false);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = () => loadOlderLogs({ beforeLoading, afterLoading });
</script>
