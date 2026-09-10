<template>
  <Search />
  <GroupedLog :name="route.params.name" :scrollable="splitColumns" />
</template>

<script lang="ts" setup>
const route = useRoute("/group/[name]");

const swarmStore = useSwarmStore();
const { customGroups } = storeToRefs(swarmStore);

const splitColumns = useSplitColumns();

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
