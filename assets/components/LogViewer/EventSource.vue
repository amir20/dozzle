<template>
  <div :class="{ 'flex h-[calc(100vh-200px)] flex-col justify-center': loading || noLogs }">
    <InfiniteLoader :onLoadMore="fetchMore" :enabled="!loadingMore && messages.length > 10" />
    <slot :messages="messages"></slot>
    <IndeterminateBar :color />
  </div>
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";

const { entity, streamSource } = $defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { messages, loadOlderLogs, isLoadingMore, opened, loading, error } = streamSource($$(entity));
const { loadingMore } = useLoggingContext();
const color = computed(() => {
  if (error.value) return "error";
  if (loading.value) return "secondary";
  if (opened.value) return "primary";
  return "error";
});

const noLogs = computed(() => messages.value.length === 0);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = async () => {
  if (!isLoadingMore.value) {
    loadingMore.value = true;
    await loadOlderLogs();
    loadingMore.value = false;
  }
};
</script>
