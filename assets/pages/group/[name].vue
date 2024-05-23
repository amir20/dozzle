<template>
  <Search />
  <GroupedLog :name="name" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const { name } = defineProps<{ name: string }>();

const swarmStore = useSwarmStore();
const { customGroups } = storeToRefs(swarmStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const group = computed(() => customGroups.value.find((g) => g.name === name));

watchEffect(() => {
  if (group.value?.name) {
    setTitle(group.value.name + " group");
  }
});
</script>
