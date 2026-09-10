<template>
  <SearchStatus :status="searchStatus" />
  <ul class="flex animate-pulse flex-col gap-4 p-4" v-if="loading || (noLogs && waitingForMoreLog && !inSearch)">
    <div class="flex flex-row gap-2" v-for="size in sizes">
      <div class="bg-base-content/50 h-3 w-40 shrink-0 rounded-full opacity-50"></div>
      <div class="bg-base-content/50 h-3 rounded-full opacity-50" :class="size"></div>
    </div>
    <span class="sr-only">Loading...</span>
  </ul>
  <div v-else-if="noLogs && !waitingForMoreLog && !inSearch" class="p-4" data-testid="no-logs">
    {{ $t("label.no-logs") }}
  </div>
  <slot :messages="messages" v-else></slot>
  <IndeterminateBar :color :intensity="streaming ? 1 : 0" v-if="!historical" />
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";
const route = useRoute();

const { entity, streamSource } = $defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { historical } = useLoggingContext();

const { messages, opened, loading, error, searchStatus } = streamSource(toRef(() => entity));

// While a search is running (or just finished), SearchStatus owns the empty
// messaging, so suppress the generic "no logs" state to avoid the false signal.
const inSearch = computed(() => searchStatus.value.active || searchStatus.value.done);

const color = computed(() => {
  if (error.value) return "error";
  if (loading.value) return "secondary";
  if (opened.value) return "primary";
  return "error";
});

// The bar reflects real throughput. `messages` is a shallow ref replaced once
// per buffer flush, so every arriving batch relights it and it fades back after
// a couple of seconds of quiet instead of implying logs are still pouring in.
const streaming = refAutoReset(false, 2000);
watch(messages, () => (streaming.value = true));

const noLogs = computed(() => messages.value.length === 0);
const waitingForMoreLog = refAutoReset(false, 3000);
watchImmediate(loading, () => (waitingForMoreLog.value = true));

defineExpose({
  clear: () => (messages.value = []),
});

if (historical.value && typeof route.query.logId === "string") {
  const targetId = route.query.logId;
  watchOnce(messages, async () => {
    await nextTick();
    document.getElementById(targetId)?.scrollIntoView({ behavior: "instant", block: "center" });
  });
}

const sizes = ref<string[]>([]);
watch(
  opened,
  (value) => {
    if (value) return;
    const sizeOptions = [
      "w-2/12",
      "w-3/12",
      "w-4/12",
      "w-5/12",
      "w-6/12",
      "w-7/12",
      "w-8/12",
      "w-9/12",
      "w-10/12",
      "w-11/12",
      "w-full",
    ];
    sizes.value = Array.from({ length: 18 }, () => sizeOptions[Math.floor(Math.random() * sizeOptions.length)]);
  },
  {
    flush: "sync",
    immediate: true,
  },
);
</script>
