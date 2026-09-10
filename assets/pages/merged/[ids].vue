<template>
  <Search />
  <MultiContainerLog :ids="ids" :scrollable="splitColumns" />
</template>

<script lang="ts" setup>
const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const route = useRoute("/merged/[ids]");

const splitColumns = useSplitColumns();

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
