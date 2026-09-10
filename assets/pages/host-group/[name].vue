<template>
  <Search />
  <HostGroupLog :name="route.params.name" :scrollable="splitColumns" />
</template>

<script lang="ts" setup>
const route = useRoute("/host-group/[name]");

const splitColumns = useSplitColumns();

watchEffect(() => {
  setTitle(route.params.name + " group");
});
</script>

<route lang="yaml">
meta:
  menu: host
</route>
