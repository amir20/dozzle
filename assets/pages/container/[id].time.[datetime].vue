<template>
  <!-- Keyed on the timestamp: vue-router reuses this component when only the
       datetime param changes, so without it a jump from one moment to another
       kept the first window on screen. The key throws away every piece of
       per-window state at once (loaded range, alert dedupe, the scroll to
       ?logId) instead of trying to reset each one. -->
  <HistoricalContainerLog
    :id
    :date
    show-title
    :scrollable="pinnedLogs.length > 0"
    v-if="currentContainer"
    :key="route.params.datetime"
  />
  <div v-else-if="ready" class="hero bg-base-200 min-h-screen">
    <div class="hero-content text-center">
      <div class="max-w-md">
        <p class="py-6 text-2xl font-bold">{{ $t("error.container-not-found") }}</p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
const route = useRoute("/container/[id].time.[datetime]");
const id = toRef(() => route.params.id);
const date = toRef(() => new Date(route.params.datetime));
const containerStore = useContainerStore();
const currentContainer = containerStore.currentContainer(id);
const { ready } = storeToRefs(containerStore);
const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

watchEffect(() => {
  if (ready.value) {
    if (currentContainer.value) {
      setTitle(currentContainer.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
<route lang="yaml">
meta:
  menu: host
</route>
