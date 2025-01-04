<template>
  <InfiniteLoader :onLoadMore="fetchMore" :enabled="!loadingMore && messages.length > 10" />
  <ul class="flex animate-pulse flex-col gap-4 p-4" v-if="loading || (noLogs && waitingForMoreLog)">
    <div class="flex flex-row gap-2" v-for="size in sizes">
      <div class="bg-base-content/50 h-3 w-40 shrink-0 rounded-full opacity-50"></div>
      <div class="bg-base-content/50 h-3 rounded-full opacity-50" :class="size"></div>
    </div>
    <span class="sr-only">Loading...</span>
  </ul>
  <div v-else-if="noLogs && !waitingForMoreLog" class="p-4">Container has no logs yet</div>
  <slot :messages="messages" v-else></slot>
  <IndeterminateBar :color />
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";

const { entity, streamSource } = $defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { messages, loadOlderLogs, isLoadingMore, opened, loading, error, eventSourceURL } = streamSource($$(entity));
const { loadingMore } = useLoggingContext();
const color = computed(() => {
  if (error.value) return "error";
  if (loading.value) return "secondary";
  if (opened.value) return "primary";
  return "error";
});

const noLogs = computed(() => messages.value.length === 0);
const waitingForMoreLog = refAutoReset(false, 3000);
watchImmediate(loading, () => (waitingForMoreLog.value = true));

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

const shuffle = (items: any[]) => {
  return items.sort(() => Math.random() - 0.5);
};

const sizes = computedWithControl(eventSourceURL, () =>
  shuffle(["w-3/5", "w-2/3", "w-9/12", "w-1/2", "w-1/3", "w-3/4"]),
);
</script>
