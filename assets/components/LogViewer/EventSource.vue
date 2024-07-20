<template>
  <InfiniteLoader :onLoadMore="fetchMore" :enabled="enabled && messages.length > 50" />
  <slot :messages="messages"></slot>
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";

const loadingMore = defineEmit<[value: boolean]>();

const { entity, streamSource } = $defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { messages, loadOlderLogs, isLoadingMore } = streamSource($$(entity));

const beforeLoading = () => loadingMore(true);
const afterLoading = () => loadingMore(false);
const enabled = ref(true);

defineExpose({
  clear: () => (messages.value = []),
});

const fetchMore = async () => {
  if (!isLoadingMore()) {
    beforeLoading();
    enabled.value = false;
    await loadOlderLogs();
    afterLoading();
    enabled.value = true;
  }
};
</script>
