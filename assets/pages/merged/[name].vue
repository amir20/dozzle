<template>
  <Search />
  <MultiContainerLog :ids="ids" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const route = useRoute("/merged/[name]");

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const ids = toRef(() => route.query.id as string[]);

watchEffect(() => {
  if (ready.value) {
    setTitle("Merged Logs");
  }
});
</script>
<route lang="yaml">
meta:
  containerMode: true
</route>
