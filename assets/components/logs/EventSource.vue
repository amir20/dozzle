<template>
  <SearchStatus :status="searchStatus" />
  <ul class="flex animate-pulse flex-col gap-4 p-4" v-if="loading || (noLogs && waitingForMoreLog && !inSearch)">
    <div class="flex flex-row gap-2" v-for="size in sizes">
      <div class="bg-base-content/50 h-3 w-40 shrink-0 rounded-full opacity-50"></div>
      <div class="bg-base-content/50 h-3 rounded-full opacity-50" :class="size"></div>
    </div>
    <span class="sr-only">Loading...</span>
  </ul>
  <EmptyState
    v-else-if="noLogs && !waitingForMoreLog && !inSearch"
    data-testid="no-logs"
    :title="$t('label.no-logs')"
    :hint="$t('label.no-logs-hint')"
  >
    <template #icon><mdi:text-box-outline class="size-5" /></template>
  </EmptyState>
  <slot :messages="messages" v-else></slot>
  <IndeterminateBar :color :intensity="streaming ? 1 : 0" v-if="!historical" />
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/logs/eventStreams";
import { HistoricalContainer } from "@/models/Container";
import { LoadMoreLogEntry } from "@/models/LogEntry";
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

// A historical view opens on a moment, and the reader has to land on that
// moment. `?logId` names the exact line when the thing that sent them here had
// one. A metric or container alert has no line at all, and without the second
// branch the view stayed wherever it loaded — the bottom of a window that can
// run to hundreds of lines, with the moment clicked thousands of pixels above
// the fold. That is what "show me the logs around it" showing no alert was.
if (historical.value) {
  const targetId = typeof route.query.logId === "string" ? route.query.logId : undefined;
  watchOnce(messages, async () => {
    await nextTick();
    const byId = targetId ? document.getElementById(targetId) : null;
    if (byId) {
      byId.scrollIntoView({ behavior: "instant", block: "center" });
      return;
    }
    const openedOn = entity instanceof HistoricalContainer ? entity.date.getTime() : undefined;
    if (openedOn === undefined) return;
    // The first real row at or past the moment, which is where an alert with no
    // line of its own is spliced in. Load-more rows carry `now` as their date
    // and pin to the ends, so they would match ahead of anything real.
    const entry = messages.value.find((m) => !(m instanceof LoadMoreLogEntry) && m.date.getTime() >= openedOn);
    if (entry) document.getElementById(entry.id.toString())?.scrollIntoView({ behavior: "instant", block: "center" });
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
