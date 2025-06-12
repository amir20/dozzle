<template>
  <HistoricalContainerLog :id :date show-title :scrollable="pinnedLogs.length > 0" v-if="currentContainer" />
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
