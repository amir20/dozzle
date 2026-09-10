<template>
  <Search />
  <ServiceLog :name="route.params.name" :scrollable="splitColumns" />
</template>

<script lang="ts" setup>
const route = useRoute("/service/[name]");

const containerStore = useContainerStore();
const { ready } = storeToRefs(containerStore);

const splitColumns = useSplitColumns();

const stackStore = useSwarmStore();
const { services } = storeToRefs(stackStore);
const service = computed(() => services.value.find((s) => s.name === route.params.name));

watchEffect(() => {
  if (ready.value) {
    if (service.value?.name) {
      setTitle(service.value.name);
    } else {
      setTitle("Not Found");
    }
  }
});
</script>
<route lang="yaml">
meta:
  menu: swarm
</route>
