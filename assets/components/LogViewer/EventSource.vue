<template>
  <InfiniteLoader :onLoadMore="fetchMore" :enabled="!loadingMore && messages.length > 10" />
  <div v-if="!opened" class="m-4 text-center">
    <span class="loading loading-ring loading-md text-primary"></span>
  </div>
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";

const { entity, streamSource } = $defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { messages, loadOlderLogs, isLoadingMore, opened } = streamSource($$(entity));
const { loadingMore } = useLoggingContext();

const enabled = ref(true);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = async () => {
  if (!isLoadingMore.value) {
    loadingMore.value = true;
    enabled.value = false;
    await loadOlderLogs();
    loadingMore.value = false;
    enabled.value = true;
  }
};
</script>
