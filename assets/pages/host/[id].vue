<template>
  <Search />
  <HostLog :id="route.params.id" :scrollable="splitColumns" />
</template>

<script lang="ts" setup>
const route = useRoute("/host/[id]");

const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const splitColumns = useSplitColumns();
const { hosts } = useHosts();
const host = computed(() => hosts.value[route.params.id]);

watchEffect(() => {
  if (ready.value) {
    if (host.value?.name) {
      setTitle(host.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
<route lang="yaml">
meta:
  menu: host
</route>
