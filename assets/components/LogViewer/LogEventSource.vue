<template>
  <infinite-loader :onLoadMore="fetchMore" :enabled="messages.length > 100"></infinite-loader>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup>
const loadingMore = defineEmit<[value: boolean]>();

const { messages, loadOlderLogs } = useLogStream();

const beforeLoading = () => loadingMore(true);
const afterLoading = () => loadingMore(false);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = () => loadOlderLogs({ beforeLoading, afterLoading });
</script>
