<template>
  <Search />
  <MultiContainerLog :ids="ids" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const route = useRoute("/merged/[ids]");

// Only the columns on screen: one hidden because this view already streams it leaves the pane full width.
const pinnedLogs = useVisiblePinnedLogs();

const ids = toRef(() => route.params.ids.split(","));

watchEffect(() => {
  if (ready.value) {
    setTitle("Merged Logs");
  }
});
</script>
<route lang="yaml">
meta:
  menu: host
</route>
