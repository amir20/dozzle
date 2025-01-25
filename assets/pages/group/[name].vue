<template>
  <Search />
  <GroupedLog :name="route.params.name" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const route = useRoute("/group/[name]");

const swarmStore = useSwarmStore();
const { customGroups } = storeToRefs(swarmStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const group = computed(() => customGroups.value.find((g) => g.name === route.params.name));

watchEffect(() => {
  if (group.value?.name) {
    setTitle(group.value.name + " group");
  }
});
</script>
<route lang="yaml">
meta:
  menu: group
</route>
