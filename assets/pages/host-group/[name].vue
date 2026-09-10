<template>
  <Search />
  <HostGroupLog :name="route.params.name" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const route = useRoute("/host-group/[name]");

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

watchEffect(() => {
  setTitle(route.params.name + " group");
});
</script>

<route lang="yaml">
meta:
  menu: host
</route>
