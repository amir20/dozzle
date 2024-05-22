<template>
  <Search />
  <GroupedLog :name="name" :scrollable="pinnedLogs.length > 0" />
</template>

<script lang="ts" setup>
const { name } = defineProps<{ name: string }>();

const containerStore = useContainerStore();
const { ready, grouped } = storeToRefs(containerStore);

const pinnedLogsStore = usePinnedLogsStore();
const { pinnedLogs } = storeToRefs(pinnedLogsStore);

const group = computed(() => grouped.value.find((g) => g.name === name));

watchEffect(() => {
  if (ready.value) {
    if (group.value?.name) {
      setTitle(group.value.name + " group");
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
