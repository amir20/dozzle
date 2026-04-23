<template>
  <Search />
  <HostGroupLog :name="route.params.name" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const route = useRoute("/host-group/[name]");

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);
const { hosts } = useHosts();

const groupHosts = computed(() => Object.values(hosts.value).filter((h) => h.group === route.params.name));

watchEffect(() => {
  if (groupHosts.value.length > 0) {
    setTitle(route.params.name + " group");
  }
});
</script>

<route lang="yaml">
meta:
  menu: host
</route>
