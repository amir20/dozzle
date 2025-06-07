<template>
  <ul class="flex animate-pulse flex-col gap-4 p-4" v-if="loading || (noLogs && waitingForMoreLog)">
    <div class="flex flex-row gap-2" v-for="size in sizes">
      <div class="bg-base-content/50 h-3 w-40 shrink-0 rounded-full opacity-50"></div>
      <div class="bg-base-content/50 h-3 rounded-full opacity-50" :class="size"></div>
    </div>
    <span class="sr-only">Loading...</span>
  </ul>
  <div v-else-if="noLogs && !waitingForMoreLog" class="p-4">
    {{ $t("label.no-logs") }}
  </div>
  <slot :messages="messages" v-else></slot>
  <IndeterminateBar :color v-if="!historical" />
</template>

<script lang="ts" setup generic="T">
import { LogStreamSource } from "@/composable/eventStreams";

const { entity, streamSource } = $defineProps<{
  streamSource: (t: Ref<T>) => LogStreamSource;
  entity: T;
}>();

const { historical } = useLoggingContext();

const { messages, opened, loading, error, eventSourceURL } = streamSource(toRef(() => entity));

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

const sizes = computedWithControl(eventSourceURL, () => {
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
  const result = [];
  const iterations = 18;
  for (let i = 0; i < iterations; i++) {
    result.push(sizeOptions[Math.floor(Math.random() * sizeOptions.length)]);
  }
  return result;
});
</script>
